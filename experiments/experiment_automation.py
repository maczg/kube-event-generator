#!/usr/bin/env python3
"""
Automated experiment runner for scheduler weight evaluation.
Generates scenarios, runs simulations, and collects results.
"""

import os
import sys
import json
import yaml
import time
import subprocess
import itertools
from datetime import datetime
from typing import Dict, List, Tuple
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class ExperimentRunner:
    def __init__(self, base_dir: str = "experiments", results_dir: str = "results"):
        self.base_dir = base_dir
        self.results_dir = results_dir
        self.experiment_id = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.experiment_results_dir = os.path.join(results_dir, f"experiment_{self.experiment_id}")
        
        # Create directories
        os.makedirs(self.base_dir, exist_ok=True)
        os.makedirs(self.experiment_results_dir, exist_ok=True)
        
        # Scheduler simulator URL
        self.scheduler_simulator_url = os.getenv("SCHEDULER_SIM_URL", "http://localhost:1212/api/v1/schedulerconfiguration")
        
        # Define experiment parameters
        self.seeds = list(range(1, 11))  # 10 seeds
        self.weight_configs = self._define_weight_configs()
        self.parameter_vectors = self._define_parameter_vectors()
        
    def _define_weight_configs(self) -> List[Dict]:
        """Define 5 different weight configurations."""
        return [
            {
                "name": "default",
                "weights": {
                    "NodeResourcesFit": 1,
                    "NodeAffinity": 1,
                    "PodTopologySpread": 1
                }
            },
            {
                "name": "resource_focused",
                "weights": {
                    "NodeResourcesFit": 10,
                    "NodeAffinity": 1,
                    "PodTopologySpread": 1
                }
            },
            {
                "name": "affinity_focused",
                "weights": {
                    "NodeResourcesFit": 1,
                    "NodeAffinity": 10,
                    "PodTopologySpread": 1
                }
            },
            {
                "name": "spread_focused",
                "weights": {
                    "NodeResourcesFit": 1,
                    "NodeAffinity": 1,
                    "PodTopologySpread": 10
                }
            },
            {
                "name": "balanced_high",
                "weights": {
                    "NodeResourcesFit": 5,
                    "NodeAffinity": 5,
                    "PodTopologySpread": 5
                }
            }
        ]
    
    def _define_parameter_vectors(self) -> List[Dict]:
        """Define 15 different parameter vectors for workload patterns."""
        vectors = []
        
        # Define ranges
        arrival_scales = [1.0, 2.0, 5.0]  # High, medium, low load
        duration_params = [(60, 1.5), (120, 2.0), (180, 2.5)]  # (scale, shape)
        cpu_params = [(100, 1.0), (200, 1.5), (400, 2.0)]  # (base, shape)
        mem_params = [(128, 1.0), (256, 1.5), (512, 2.0)]  # (scale, shape)
        
        # Generate combinations (select 15)
        all_combos = list(itertools.product(
            arrival_scales, duration_params, cpu_params, mem_params
        ))
        
        # Select 15 diverse combinations
        selected_indices = [0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28]
        
        for idx, combo_idx in enumerate(selected_indices[:15]):
            if combo_idx < len(all_combos):
                arr, (dur_scale, dur_shape), (cpu_base, cpu_shape), (mem_scale, mem_shape) = all_combos[combo_idx]
                
                vectors.append({
                    "name": f"workload_{idx+1}",
                    "numPodEvents": 100,
                    "arrivalScale": arr,
                    "arrivalScaleFactor": 5.0,
                    "durationScale": dur_scale,
                    "durationShape": dur_shape,
                    "durationScaleFactor": 10.0,
                    "podCpuShape": cpu_shape,
                    "podCpuFactor": cpu_base / 1000.0,  # Convert to cores
                    "podMemScale": mem_scale,
                    "podMemShape": mem_shape,
                    "podMemFactor": 1.0
                })
        
        return vectors
    
    def generate_config_file(self, param_vector: Dict, output_path: str, include_cluster: bool = True):
        """Generate a KEG configuration file for given parameters.
        
        Args:
            param_vector: Parameters for the scenario
            output_path: Where to save the config file
            include_cluster: Whether to include cluster spec in the config
        """
        # Create generation config with embedded cluster parameters
        generation_config = param_vector.copy()
        
        # Add cluster configuration if requested
        if include_cluster:
            cluster_config = self.get_cluster_config_for_workload(param_vector)
            generation_config["cluster"] = {
                "generate": True,
                "numNodes": cluster_config['num_nodes'],
                "cpuPerNode": cluster_config['cpu_per_node'],
                "memoryPerNode": cluster_config['memory_per_node'],
                "podsPerNode": cluster_config['pods_per_node'],
                "zones": ["zone-a", "zone-b", "zone-c"]
            }
            logger.info(f"Added cluster config: {cluster_config['num_nodes']} nodes with {cluster_config['cpu_per_node']} CPU each")
        
        config = {
            "scenarioName": param_vector["name"],  # Top level, not inside scenario
            "outputDir": "scenarios",              # KEG uses this for output
            "generation": generation_config,       # Generation params with cluster config
            "kubernetes": {
                "namespace": "default",
                "requestTimeout": "30s"
            },
            "scheduler": {
                "simulatorUrl": self.scheduler_simulator_url,
                "httpTimeout": "10s"
            },
            "output": {
                "saveMetrics": True,
                "outputDir": "results",
                "format": "csv"
            }
        }
        
        with open(output_path, 'w') as f:
            yaml.dump(config, f, default_flow_style=False)
    
    def generate_cluster_nodes(self, num_nodes: int = 3, cpu_per_node: str = "2", 
                              memory_per_node: str = "4Gi", pods_per_node: int = 50) -> List[Dict]:
        """Generate cluster node definitions for KWOK simulation.
        
        Args:
            num_nodes: Number of nodes to create
            cpu_per_node: CPU capacity per node (e.g., "2", "4")
            memory_per_node: Memory capacity per node (e.g., "4Gi", "8Gi")
            pods_per_node: Maximum pods per node
        """
        nodes = []
        zones = ["zone-a", "zone-b", "zone-c"]
        
        # Calculate allocatable resources (95% of capacity for CPU, 87.5% for memory)
        cpu_value = float(cpu_per_node)
        allocatable_cpu = f"{int(cpu_value * 950)}m"  # 95% of capacity in millicores
        
        memory_value = float(memory_per_node.replace('Gi', ''))
        allocatable_memory = f"{memory_value * 0.875:.1f}Gi"  # 87.5% of capacity
        
        for i in range(num_nodes):
            node_name = f"node-{i+1}"
            zone = zones[i % len(zones)]  # Cycle through zones
            
            nodes.append({
                "metadata": {
                    "name": node_name,
                    "labels": {
                        "kubernetes.io/hostname": node_name,
                        "node.kubernetes.io/instance-type": "standard",
                        "topology.kubernetes.io/zone": zone,
                        "type": "kwok"  # Mark as KWOK node
                    }
                },
                "status": {
                    "capacity": {
                        "cpu": cpu_per_node,
                        "memory": memory_per_node,
                        "pods": str(pods_per_node)
                    },
                    "allocatable": {
                        "cpu": allocatable_cpu,
                        "memory": allocatable_memory,
                        "pods": str(pods_per_node)
                    }
                }
            })
        
        return nodes
    
    def get_cluster_config_for_workload(self, param_vector: Dict) -> Dict:
        """Generate appropriate cluster configuration based on workload parameters."""
        # Extract workload characteristics
        num_pods = param_vector.get('numPodEvents', 100)
        
        # For pod resources, we need to consider the distribution parameters
        # The actual values will be generated from Weibull distribution
        # We'll estimate based on scale factors
        cpu_factor = param_vector.get('podCpuFactor', 0.1)  # in cores
        mem_scale = param_vector.get('podMemScale', 100)  # in Mi
        
        # Estimate average resource per pod (considering distribution mean)
        # For Weibull, mean ≈ scale * Γ(1 + 1/shape)
        # Simplified: use scale * 1.5 as rough estimate
        avg_cpu_per_pod = cpu_factor * 1.5
        avg_memory_per_pod = (mem_scale * 1.5) / 1024.0  # Convert to Gi
        
        # Calculate total resources needed (with 50% headroom)
        total_cpu_needed = num_pods * avg_cpu_per_pod * 1.5
        total_memory_needed = num_pods * avg_memory_per_pod * 1.5
        
        # Determine node size and count
        if total_cpu_needed <= 6:  # Small cluster
            node_cpu = 2
            node_memory = 4
            num_nodes = 3
        elif total_cpu_needed <= 12:  # Medium cluster
            node_cpu = 4
            node_memory = 8
            num_nodes = 3
        else:  # Large cluster
            node_cpu = 4
            node_memory = 8
            num_nodes = max(3, min(10, int(total_cpu_needed / 4) + 1))
        
        # Ensure we have enough pod capacity
        pods_per_node = max(50, int(num_pods / num_nodes * 1.5))
        
        return {
            "num_nodes": num_nodes,
            "cpu_per_node": node_cpu,
            "memory_per_node": node_memory,
            "pods_per_node": pods_per_node
        }

    def generate_scenario_with_weights(self, base_scenario_path: str, weight_config: Dict, output_path: str):
        """Modify a scenario to include scheduler weight change events."""
        with open(base_scenario_path, 'r') as f:
            scenario = yaml.safe_load(f)

        # Don't modify cluster nodes - they should already be in the base scenario
        # generated from the config file

        # Add scheduler event at the beginning
        if 'events' not in scenario:
            scenario['events'] = {}
        
        # Check if scheduler key exists and is not null
        if 'scheduler' not in scenario['events'] or scenario['events']['scheduler'] is None:
            scenario['events']['scheduler'] = []
        
        # Insert weight change event at time 0
        scenario['events']['scheduler'].insert(0, {
            'name': f'set_weights_{weight_config["name"]}',
            'after': '0s',
            'weights': weight_config['weights']
        })
        
        with open(output_path, 'w') as f:
            yaml.dump(scenario, f, default_flow_style=False)
    
    def ensure_clean_cluster_state(self, scenario_path: str) -> bool:
        """Ensure cluster is in a clean, consistent state before running experiment."""
        try:
            logger.info("Ensuring clean cluster state...")
            
            # Step 1: Delete all non-system pods
            logger.info("Cleaning up existing pods...")
            namespaces_cmd = ["kubectl", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}"]
            result = subprocess.run(namespaces_cmd, capture_output=True, text=True, timeout=30)
            
            if result.returncode == 0:
                namespaces = result.stdout.strip().split()
                system_namespaces = {"kube-system", "kube-public", "kube-node-lease", "local-path-storage"}
                
                for ns in namespaces:
                    if ns not in system_namespaces:
                        delete_cmd = ["kubectl", "delete", "pods", "--all", "-n", ns, "--force", "--grace-period=0"]
                        subprocess.run(delete_cmd, capture_output=True, text=True, timeout=60)
            
            # Step 2: Reset scheduler configuration to default
            if not self.reset_scheduler_config():
                logger.warning("Failed to reset scheduler config, continuing anyway")
            
            # Step 3: For KWOK environments, recreate nodes from scenario
            is_kwok = self.is_kwok_environment()
            logger.info(f"KWOK environment check: {is_kwok}")
            if is_kwok:
                logger.info("KWOK environment detected, recreating nodes from scenario...")
                
                # Delete all existing nodes
                delete_nodes_cmd = ["kubectl", "delete", "nodes", "--all"]
                subprocess.run(delete_nodes_cmd, capture_output=True, text=True, timeout=60)
                time.sleep(5)
                
                # Load scenario to get node definitions
                with open(scenario_path, 'r') as f:
                    scenario = yaml.safe_load(f)
                
                if 'cluster' in scenario and scenario['cluster'] and 'nodes' in scenario['cluster']:
                    nodes = scenario['cluster']['nodes']
                    logger.info(f"Creating {len(nodes)} nodes from scenario")
                    
                    for node in nodes:
                        # Create node manifest
                        node_manifest = {
                            "apiVersion": "v1",
                            "kind": "Node",
                            "metadata": node.get('metadata', {}),
                            "status": node.get('status', {})
                        }
                        
                        # Apply node
                        create_cmd = ["kubectl", "apply", "-f", "-"]
                        process = subprocess.Popen(create_cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                        stdout, stderr = process.communicate(input=yaml.dump(node_manifest))
                        
                        if process.returncode != 0:
                            logger.error(f"Failed to create node: {stderr}")
                            return False
                else:
                    logger.warning("No cluster definition found in scenario, creating default nodes")
                    # Fallback to standard nodes if scenario has no cluster definition
                    standard_nodes = self.generate_cluster_nodes(num_nodes=3, cpu_per_node="2", memory_per_node="4Gi")
                    for node in standard_nodes:
                        node_manifest = {
                            "apiVersion": "v1",
                            "kind": "Node",
                            "metadata": node.get('metadata', {}),
                            "status": node.get('status', {})
                        }
                        create_cmd = ["kubectl", "apply", "-f", "-"]
                        process = subprocess.Popen(create_cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                        stdout, stderr = process.communicate(input=yaml.dump(node_manifest))
                        if process.returncode != 0:
                            logger.error(f"Failed to create fallback node: {stderr}")
                            return False
            
            # Step 4: Wait for nodes to be ready
            if not self.wait_for_nodes_ready():
                logger.error("Nodes did not become ready")
                return False
            
            # Step 5: Validate cluster state
            if not self.validate_cluster_state():
                logger.error("Cluster validation failed")
                return False
            
            logger.info("Cluster is in clean state and ready for experiment")
            return True
            
        except Exception as e:
            logger.error(f"Error ensuring clean cluster state: {str(e)}")
            return False
    
    def reset_scheduler_config(self) -> bool:
        """Reset scheduler configuration to default state."""
        try:
            logger.info("Resetting scheduler configuration...")
            
            # Default scheduler config with standard plugin weights
            default_config = {
                "profiles": [{
                    "schedulerName": "default-scheduler",
                    "plugins": {
                        "score": {
                            "enabled": [
                                {"name": "NodeResourcesFit", "weight": 1},
                                {"name": "ImageLocality", "weight": 1},
                                {"name": "InterPodAffinity", "weight": 1},
                                {"name": "NodeAffinity", "weight": 1},
                                {"name": "PodTopologySpread", "weight": 2},
                                {"name": "TaintToleration", "weight": 1}
                            ]
                        }
                    }
                }]
            }
            
            # If scheduler simulator is available, reset it
            if self.scheduler_simulator_url:
                try:
                    import requests
                    response = requests.put(
                        self.scheduler_simulator_url,
                        json=default_config,
                        timeout=10
                    )
                    if response.status_code != 200:
                        logger.warning(f"Scheduler reset returned status {response.status_code}")
                        return False
                except ImportError:
                    logger.warning("requests module not available, skipping scheduler reset")
                    return False
                except Exception as e:
                    logger.warning(f"Failed to reset scheduler config via API: {str(e)}")
                    return False
            
            return True
            
        except Exception as e:
            logger.error(f"Error resetting scheduler config: {str(e)}")
            return False
    
    def is_kwok_environment(self) -> bool:
        """Check if running in KWOK environment."""
        try:
            # Check if API server is running on port 3131 (KWOK default)
            cmd = ["kubectl", "config", "view", "-o", "jsonpath={.clusters[*].cluster.server}"]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            
            if result.returncode == 0 and "3131" in result.stdout:
                return True
                
            # Check if any node has kwok provider label
            cmd = ["kubectl", "get", "nodes", "-o", "jsonpath={.items[*].metadata.labels.type}"]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            
            if result.returncode == 0 and "kwok" in result.stdout:
                return True
            
            # Also check for kwok annotation
            cmd = ["kubectl", "get", "nodes", "-o", "jsonpath={.items[*].metadata.annotations}"]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            
            if result.returncode == 0 and "kwok" in result.stdout.lower():
                return True
                
        except Exception:
            pass
        
        return False
    
    def wait_for_nodes_ready(self, timeout: int = 120) -> bool:
        """Wait for all nodes to become ready."""
        logger.info("Waiting for nodes to become ready...")
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            cmd = ["kubectl", "get", "nodes", "-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}"]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            
            if result.returncode == 0:
                statuses = result.stdout.strip().split()
                if statuses and all(status == "True" for status in statuses):
                    logger.info("All nodes are ready")
                    return True
            
            time.sleep(5)
        
        return False
    
    def validate_cluster_state(self) -> bool:
        """Validate that cluster is in expected clean state."""
        try:
            logger.info("Validating cluster state...")
            
            # Check no non-system pods exist
            cmd = ["kubectl", "get", "pods", "--all-namespaces", "-o", "json"]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
            
            if result.returncode == 0:
                pods_data = json.loads(result.stdout)
                system_namespaces = {"kube-system", "kube-public", "kube-node-lease", "local-path-storage"}
                
                non_system_pods = [
                    pod for pod in pods_data.get('items', [])
                    if pod['metadata']['namespace'] not in system_namespaces
                ]
                
                if non_system_pods:
                    logger.warning(f"Found {len(non_system_pods)} non-system pods")
                    for pod in non_system_pods:
                        logger.warning(f"  - {pod['metadata']['namespace']}/{pod['metadata']['name']}")
                    return False
            
            # Check nodes are ready
            cmd = ["kubectl", "get", "nodes", "-o", "json"]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
            
            if result.returncode == 0:
                nodes_data = json.loads(result.stdout)
                nodes = nodes_data.get('items', [])
                
                if not nodes:
                    logger.error("No nodes found in cluster")
                    return False
                
                for node in nodes:
                    conditions = node.get('status', {}).get('conditions', [])
                    ready_condition = next((c for c in conditions if c['type'] == 'Ready'), None)
                    
                    if not ready_condition or ready_condition['status'] != 'True':
                        logger.error(f"Node {node['metadata']['name']} is not ready")
                        return False
            
            logger.info("Cluster validation passed")
            return True
            
        except Exception as e:
            logger.error(f"Error validating cluster: {str(e)}")
            return False
    
    def reset_cluster(self) -> bool:
        """Legacy reset function - kept for backward compatibility."""
        # This is kept for backward compatibility
        # In the new flow, we'll call ensure_clean_cluster_state with the scenario path
        logger.warning("Using legacy reset_cluster - should use ensure_clean_cluster_state instead")
        try:
            # Basic reset without scenario-specific node creation
            reset_cmd = ["./bin/keg", "cluster", "reset", "--force"]
            result = subprocess.run(reset_cmd, capture_output=True, text=True, timeout=60)
            
            if result.returncode != 0:
                logger.warning(f"Cluster reset warning: {result.stderr}")
            
            time.sleep(5)
            return True
        except Exception as e:
            logger.error(f"Error in legacy reset: {str(e)}")
            return False
    
    def verify_cluster_nodes(self) -> bool:
        """Verify that cluster nodes are ready."""
        try:
            logger.info("Verifying cluster nodes are ready...")
            
            # Verify all nodes are ready
            max_retries = 12  # 2 minutes max
            for attempt in range(max_retries):
                verify_cmd = ["kubectl", "get", "nodes", "-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}"]
                result = subprocess.run(verify_cmd, capture_output=True, text=True)
                
                if result.returncode == 0:
                    statuses = result.stdout.strip().split()
                    if statuses and all(status == "True" for status in statuses):
                        logger.info("All nodes are ready")
                        return True
                
                logger.info(f"Waiting for nodes to be ready... (attempt {attempt + 1}/{max_retries})")
                time.sleep(10)
            
            logger.error("Nodes did not become ready within timeout")
            return False
            
        except Exception as e:
            logger.error(f"Error verifying cluster nodes: {str(e)}")
            return False
    
    def cleanup_cluster_resources(self) -> bool:
        """Clean up any remaining pods or resources."""
        try:
            logger.info("Cleaning up cluster resources...")
            
            # Delete all pods in default namespace
            cleanup_cmd = ["kubectl", "delete", "pods", "--all", "-n", "default", "--force", "--grace-period=0"]
            subprocess.run(cleanup_cmd, capture_output=True, text=True, timeout=60)
            
            # Wait for cleanup
            time.sleep(5)
            
            return True
            
        except Exception as e:
            logger.warning(f"Error during cleanup: {str(e)}")
            return False
    
    def run_experiment(self, param_name: str, weight_name: str, seed: int) -> bool:
        """Run a single experiment with proper cluster management."""
        run_name = f"{param_name}_{weight_name}_seed{seed}"
        run_dir = os.path.join(self.experiment_results_dir, run_name)
        
        try:
            # Find the scenario file in scenarios directory
            scenario_path = os.path.join("scenarios", f"{param_name}_seed{seed}_{weight_name}.yaml")
            
            if not os.path.exists(scenario_path):
                logger.error(f"Scenario file not found: {scenario_path}")
                return False
            
            # Step 1: Ensure clean cluster state before experiment
            logger.info(f"Preparing cluster for experiment: {run_name}")
            if not self.ensure_clean_cluster_state(scenario_path):
                logger.error(f"Failed to prepare cluster for {run_name}")
                return False
            
            # Step 2: Run simulation (without --cluster-reset since we already reset)
            cmd = [
                "./bin/keg", "simulation", "start",
                "-s", scenario_path,
                "--output-dir", run_dir,
                "--save-metrics"
                # Note: removed --cluster-reset flag since we handle it manually now
            ]
            
            logger.info(f"Running simulation: {run_name}")
            logger.info(f"Command: {' '.join(cmd)}")
            
            # Run with better output handling
            process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            stdout_lines = []
            stderr_lines = []
            
            try:
                # Wait for completion with timeout
                stdout, stderr = process.communicate(timeout=600)  # 10 minute timeout
                
                if stdout:
                    stdout_lines = stdout.splitlines()
                    for line in stdout_lines[-20:]:  # Log last 20 lines
                        logger.debug(f"[KEG] {line}")
                
                if stderr:
                    stderr_lines = stderr.splitlines()
                    for line in stderr_lines:
                        logger.warning(f"[KEG ERROR] {line}")
                
                if process.returncode != 0:
                    logger.error(f"Simulation failed for {run_name} with return code {process.returncode}")
                    if stderr_lines:
                        logger.error(f"Last error lines: {stderr_lines[-10:]}")
                    return False
                    
            except subprocess.TimeoutExpired:
                logger.error(f"Simulation timed out after 600 seconds")
                process.terminate()
                time.sleep(5)
                if process.poll() is None:
                    process.kill()
                return False
            
            # Step 3: Save weight configuration metadata
            os.makedirs(run_dir, exist_ok=True)
            with open(os.path.join(run_dir, "weight_config.json"), 'w') as f:
                json.dump({
                    "weight_name": weight_name,
                    "weights": next(w['weights'] for w in self.weight_configs if w['name'] == weight_name),
                    "parameter_vector": param_name,
                    "seed": seed,
                    "run_name": run_name
                }, f, indent=2)
            
            logger.info(f"Experiment {run_name} completed successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error running {run_name}: {str(e)}")
            return False
    
    def verify_environment(self) -> bool:
        """Verify that the environment is ready for experiments."""
        try:
            # Check if KEG binary exists
            if not os.path.exists("./bin/keg"):
                logger.error("KEG binary not found. Run 'make build' first.")
                return False
            
            # Check if kubectl is available
            result = subprocess.run(["kubectl", "version", "--client"], capture_output=True, text=True)
            if result.returncode != 0:
                logger.error("kubectl not available")
                return False
            
            # Check if KWOK cluster is running
            result = subprocess.run(["kubectl", "get", "nodes"], capture_output=True, text=True)
            if result.returncode != 0:
                logger.warning("KWOK cluster not accessible. Starting cluster...")
                # Try to start the cluster
                start_result = subprocess.run(
                    ["docker-compose", "-f", "docker/docker-compose.yaml", "up", "-d"],
                    capture_output=True, text=True
                )
                if start_result.returncode != 0:
                    logger.error("Failed to start KWOK cluster")
                    return False
                time.sleep(15)  # Wait for cluster to be ready
            
            logger.info("Environment verification successful")
            return True
            
        except Exception as e:
            logger.error(f"Environment verification failed: {str(e)}")
            return False
    
    def save_progress_checkpoint(self, completed: int, failed: int, current_batch: int):
        """Save progress checkpoint for recovery."""
        checkpoint = {
            "experiment_id": self.experiment_id,
            "timestamp": datetime.now().isoformat(),
            "completed": completed,
            "failed": failed,
            "current_batch": current_batch
        }
        
        checkpoint_path = os.path.join(self.experiment_results_dir, "progress_checkpoint.json")
        with open(checkpoint_path, 'w') as f:
            json.dump(checkpoint, f, indent=2)
    
    def run_all_experiments(self, batch_size: int = 5, delay_between_batches: int = 60, 
                          max_retries_per_experiment: int = 2):
        """Run all experiments in batches with robust error handling."""
        
        # Verify environment first
        if not self.verify_environment():
            logger.error("Environment verification failed. Cannot proceed.")
            return 0, 0
        
        total_experiments = len(self.seeds) * len(self.weight_configs) * len(self.parameter_vectors)
        logger.info(f"Starting {total_experiments} experiments")
        logger.info(f"Batch size: {batch_size}, Delay between batches: {delay_between_batches}s")
        
        # First, generate all scenarios
        logger.info("Generating scenarios...")
        scenarios_generated, scenarios_failed = self.generate_all_scenarios()
        
        if scenarios_failed > 0:
            logger.warning(f"Failed to generate {scenarios_failed} scenarios")
        
        if scenarios_generated == 0:
            logger.error("No scenarios were generated successfully")
            return 0, total_experiments
        
        # Run experiments in batches
        logger.info("Running experiments...")
        experiment_queue = [
            (param['name'], weight['name'], seed)
            for param in self.parameter_vectors
            for weight in self.weight_configs
            for seed in self.seeds
        ]
        
        completed = 0
        failed = 0
        batch_number = 0
        
        for i in range(0, len(experiment_queue), batch_size):
            batch = experiment_queue[i:i+batch_size]
            batch_number += 1
            
            logger.info(f"Running batch {batch_number} ({len(batch)} experiments)")
            logger.info(f"Progress: {completed + failed}/{total_experiments} experiments processed")
            
            batch_start_time = time.time()
            
            for param_name, weight_name, seed in batch:
                experiment_start_time = time.time()
                
                # Try the experiment with retries
                success = False
                for attempt in range(max_retries_per_experiment):
                    if attempt > 0:
                        logger.info(f"Retrying experiment {param_name}_{weight_name}_seed{seed} (attempt {attempt + 1})")
                        time.sleep(30)  # Wait before retry
                    
                    if self.run_experiment(param_name, weight_name, seed):
                        success = True
                        break
                    else:
                        logger.warning(f"Experiment attempt {attempt + 1} failed")
                
                if success:
                    completed += 1
                    experiment_duration = time.time() - experiment_start_time
                    logger.info(f"Experiment completed in {experiment_duration:.1f}s")
                else:
                    failed += 1
                    logger.error(f"Experiment {param_name}_{weight_name}_seed{seed} failed after {max_retries_per_experiment} attempts")
                
                # Save progress checkpoint
                self.save_progress_checkpoint(completed, failed, batch_number)
                
                # Small delay between experiments within batch
                time.sleep(5)
            
            batch_duration = time.time() - batch_start_time
            logger.info(f"Batch {batch_number} completed in {batch_duration:.1f}s")
            
            # Longer delay between batches to let system stabilize
            if i + batch_size < len(experiment_queue):
                logger.info(f"Waiting {delay_between_batches}s before next batch...")
                logger.info(f"Current success rate: {completed/(completed + failed)*100:.1f}%")
                time.sleep(delay_between_batches)
        
        logger.info(f"All experiments completed!")
        logger.info(f"Successful: {completed}, Failed: {failed}")
        logger.info(f"Success rate: {completed/total_experiments*100:.1f}%")
        
        # Save final experiment metadata
        metadata = {
            "experiment_id": self.experiment_id,
            "total_experiments": total_experiments,
            "completed": completed,
            "failed": failed,
            "success_rate": completed/total_experiments,
            "seeds": self.seeds,
            "weight_configs": self.weight_configs,
            "parameter_vectors": self.parameter_vectors,
            "batch_size": batch_size,
            "delay_between_batches": delay_between_batches,
            "max_retries_per_experiment": max_retries_per_experiment,
            "completion_time": datetime.now().isoformat()
        }
        
        with open(os.path.join(self.experiment_results_dir, "experiment_metadata.json"), 'w') as f:
            json.dump(metadata, f, indent=2)
        
        # Generate summary report
        self.generate_summary_report(completed, failed)
        
        return completed, failed
    
    def generate_summary_report(self, completed: int, failed: int):
        """Generate a summary report of the experiment run."""
        report_path = os.path.join(self.experiment_results_dir, "experiment_summary.txt")
        
        total = completed + failed
        success_rate = completed / total * 100 if total > 0 else 0
        
        with open(report_path, 'w') as f:
            f.write("EXPERIMENT RUN SUMMARY\n")
            f.write("=" * 50 + "\n\n")
            f.write(f"Experiment ID: {self.experiment_id}\n")
            f.write(f"Completion Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
            f.write(f"Total Experiments: {total}\n")
            f.write(f"Successful: {completed}\n")
            f.write(f"Failed: {failed}\n")
            f.write(f"Success Rate: {success_rate:.1f}%\n\n")
            
            f.write("CONFIGURATION:\n")
            f.write(f"- Seeds: {len(self.seeds)} ({min(self.seeds)} to {max(self.seeds)})\n")
            f.write(f"- Weight Configurations: {len(self.weight_configs)}\n")
            f.write(f"- Parameter Vectors: {len(self.parameter_vectors)}\n\n")
            
            f.write("WEIGHT CONFIGURATIONS:\n")
            for config in self.weight_configs:
                f.write(f"- {config['name']}: {config['weights']}\n")
            
            f.write(f"\nNext Steps:\n")
            f.write(f"1. Run analysis: keg analyze compare -s {self.experiment_results_dir}/*\n")
            f.write(f"2. Generate report: python analyzer/compare_simulations.py --simulations $(ls -d {self.experiment_results_dir}/*/) --output {self.experiment_results_dir}/analysis\n")
        
        logger.info(f"Summary report saved to: {report_path}")
    
    def generate_all_scenarios(self) -> Tuple[int, int]:
        """Generate all scenario files without running simulations."""
        logger.info("Generating all scenario files...")
        
        total_scenarios = len(self.seeds) * len(self.weight_configs) * len(self.parameter_vectors)
        logger.info(f"Will generate {total_scenarios} scenario files")
        
        scenarios_generated = 0
        scenarios_failed = 0
        
        # Ensure scenarios directory exists
        os.makedirs("scenarios", exist_ok=True)
        
        # Generate scenarios
        for param_vector in self.parameter_vectors:
            logger.info(f"Generating scenarios for parameter vector: {param_vector['name']}")
            
            # Generate scenarios for each seed
            for seed in self.seeds:
                # Create a unique scenario name with seed
                scenario_name = f"{param_vector['name']}_seed{seed}"
                
                # Create config with cluster spec embedded
                config_path = os.path.join(self.base_dir, f"config_{scenario_name}.yaml")
                seed_param_vector = param_vector.copy()
                seed_param_vector['name'] = scenario_name
                self.generate_config_file(seed_param_vector, config_path, include_cluster=True)
                
                # Generate scenario using KEG
                cmd = [
                    "./bin/keg", "scenario", "generate",
                    "-c", config_path,
                    "--seed", str(seed)
                ]
                
                # Expected output path based on KEG's behavior - scenarios folder
                expected_scenario_path = os.path.join("scenarios", f"{scenario_name}.yaml")
                
                # Ensure scenarios directory exists
                os.makedirs("scenarios", exist_ok=True)
                
                logger.info(f"Generating base scenario: {scenario_name}")
                result = subprocess.run(cmd, capture_output=True, text=True)
                
                if result.returncode != 0:
                    logger.error(f"Failed to generate scenario: {result.stderr}")
                    scenarios_failed += 1
                    continue
                
                # Verify scenario was created
                if not os.path.exists(expected_scenario_path):
                    logger.error(f"Scenario file not found at expected location: {expected_scenario_path}")
                    logger.debug(f"KEG output: {result.stdout}")
                    scenarios_failed += 1
                    continue
                
                logger.info(f"Base scenario generated: {expected_scenario_path}")
                
                # Create versions with different scheduler weights
                for weight_config in self.weight_configs:
                    # Final scenario path in scenarios directory for consistency
                    weighted_scenario_path = os.path.join(
                        "scenarios", 
                        f"{param_vector['name']}_seed{seed}_{weight_config['name']}.yaml"
                    )
                    
                    try:
                        self.generate_scenario_with_weights(expected_scenario_path, weight_config, weighted_scenario_path)
                        scenarios_generated += 1
                        logger.debug(f"Generated weighted scenario: {weighted_scenario_path}")
                    except Exception as e:
                        logger.error(f"Failed to generate weighted scenario: {str(e)}")
                        scenarios_failed += 1
                
                # Clean up temporary config file
                try:
                    os.remove(config_path)
                except Exception:
                    pass
        
        logger.info(f"Scenario generation complete!")
        logger.info(f"Generated: {scenarios_generated}, Failed: {scenarios_failed}")
        
        # Save scenario generation metadata
        metadata = {
            "experiment_id": self.experiment_id,
            "generation_mode": "scenarios_only",
            "total_scenarios_expected": total_scenarios,
            "scenarios_generated": scenarios_generated,
            "scenarios_failed": scenarios_failed,
            "seeds": self.seeds,
            "weight_configs": [{"name": w["name"], "weights": w["weights"]} for w in self.weight_configs],
            "parameter_vectors": [{"name": p["name"]} for p in self.parameter_vectors],
            "completion_time": datetime.now().isoformat()
        }
        
        metadata_path = os.path.join(self.base_dir, f"scenario_generation_metadata_{self.experiment_id}.json")
        with open(metadata_path, 'w') as f:
            json.dump(metadata, f, indent=2)
        
        logger.info(f"Metadata saved to: {metadata_path}")
        
        # List generated files
        scenario_files = [f for f in os.listdir(self.base_dir) if f.startswith('scenario_') and f.endswith('.yaml')]
        logger.info(f"Generated scenario files ({len(scenario_files)}):")
        for f in sorted(scenario_files)[:10]:  # Show first 10
            logger.info(f"  - {f}")
        if len(scenario_files) > 10:
            logger.info(f"  ... and {len(scenario_files) - 10} more")
        
        return scenarios_generated, scenarios_failed


def main():
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Run scheduler weight experiments",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Run with smaller batches and longer delays for stability
  python experiments/experiment_automation.py --batch-size 3 --delay 120

  # Quick test run with fewer parameters
  python experiments/experiment_automation.py --test-mode

  # Resume failed experiments
  python experiments/experiment_automation.py --retry-failed

Note: This will run 750 experiments total (10 seeds × 5 weights × 15 parameters).
Each experiment includes cluster reset, creation, and simulation.
Estimated time: 8-12 hours depending on configuration.
        """
    )
    
    parser.add_argument("--batch-size", type=int, default=5, 
                       help="Number of experiments per batch (default: 5)")
    parser.add_argument("--delay", type=int, default=60, 
                       help="Delay between batches in seconds (default: 60)")
    parser.add_argument("--max-retries", type=int, default=2,
                       help="Maximum retries per failed experiment (default: 2)")
    parser.add_argument("--base-dir", default="experiments", 
                       help="Base directory for experiments")
    parser.add_argument("--results-dir", default="results", 
                       help="Results directory")
    parser.add_argument("--test-mode", action="store_true",
                       help="Run with reduced parameters for testing (2 seeds, 2 weights, 3 params)")
    parser.add_argument("--dry-run", action="store_true",
                       help="Generate scenarios but don't run simulations")
    parser.add_argument("--continue-from", type=str,
                       help="Continue from specific experiment ID")
    parser.add_argument("--debug", action="store_true",
                       help="Enable debug logging")
    
    args = parser.parse_args()
    
    # Set logging level based on debug flag
    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)
        logger.info("Debug logging enabled")
    
    runner = ExperimentRunner(args.base_dir, args.results_dir)
    
    # Test mode - reduce parameters for quick testing
    if args.test_mode:
        logger.info("Running in test mode with reduced parameters")
        runner.seeds = [1, 2]
        runner.weight_configs = runner.weight_configs[:2]  # First 2 weight configs
        runner.parameter_vectors = runner.parameter_vectors[:3]  # First 3 parameter vectors
        
        total = len(runner.seeds) * len(runner.weight_configs) * len(runner.parameter_vectors)
        logger.info(f"Test mode: {total} experiments instead of 750")
    
    if args.dry_run:
        logger.info("Dry run mode - generating scenarios only")
        completed, failed = runner.generate_all_scenarios()
    else:
        completed, failed = runner.run_all_experiments(
            batch_size=args.batch_size, 
            delay_between_batches=args.delay,
            max_retries_per_experiment=args.max_retries
        )
    
    print(f"\n{'='*60}")
    print(f"EXPERIMENT RUN SUMMARY")
    print(f"{'='*60}")
    print(f"Experiment ID: {runner.experiment_id}")
    print(f"Completed: {completed}")
    print(f"Failed: {failed}")
    
    if completed + failed > 0:
        success_rate = completed / (completed + failed) * 100
        print(f"Success Rate: {success_rate:.1f}%")
    
    print(f"Results Directory: {runner.experiment_results_dir}")
    print(f"{'='*60}")
    
    if completed > 0:
        print(f"\nNext steps:")
        print(f"1. Analyze results:")
        print(f"   keg analyze compare -s {runner.experiment_results_dir}/*")
        print(f"2. Generate comprehensive report:")
        print(f"   python analyzer/compare_simulations.py \\")
        print(f"     --simulations $(ls -d {runner.experiment_results_dir}/*/) \\")
        print(f"     --output {runner.experiment_results_dir}/analysis")


if __name__ == "__main__":
    main()