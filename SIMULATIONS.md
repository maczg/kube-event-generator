| sim-id  	 | Weights                                                                                                                                                                                   	 | cluster                              	 | events 	 | event_sentinel                     	 | defragmentation_index 	 |
|-----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------|----------|--------------------------------------|-------------------------|
| default 	 | NRF: 1<br>NRBA: 1<br>at time 0s                                                                              	                                                                              | node-1 4Cpu 32GB<br>node-2 4Cpu 32GB 	 | 100    	 | cpu_ref: 903.79<br>mem_ref: 446.06 	 | 250.46              	   |
| A       	 | **Time 0s**<br>NRF: 1<br>NRBA: 1<br><br>**Time 30s**<br>NRF: 20<br>NRBA: 1<br>**Time 1m**<br>NRF: 1<br>NRBA: 20<br>**Time 1m**<br>NRF: 20<br>NRBA: 1<br>**Time 2m**<br>NRF: 1<br>NRBA: 20 	 | node-1 4Cpu 32GB<br>node-2 4Cpu 32GB 	 | 100    	 | cpu_ref: 903.79<br>mem_ref: 446.06 	 | 228.81              	   |
| B       	 | **Time 0s**<br>NRF: 1<br>NRBA: 20                                                                                                                                                         	 | node-1 4Cpu 32GB<br>node-2 4Cpu 32GB 	 | 	        | 	                                    | 213.34                  |
| c    	    | **Time 0s**<br>NRF: 1<br>NRBA: 20                                                                                                                                                         	 | node-1 4Cpu 32GB<br>node-2 4Cpu 32GB 	 | 	        | 	                                    | 208.41                  |

# Simulations 

### Parameters

#### randoms
- Seed

#### loads
- Number of pods: 10, 20, 40, 80, 100
- Average duration determined by [durationScale, durationShape, durationScaleFactor] 