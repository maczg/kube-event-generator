package cluster

import (
	"github.com/maczg/kube-event-generator/pkg/cache"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/utils"
	"github.com/spf13/cobra"
	"sort"
)

var log = logger.NewLogger(logger.LevelInfo, "cluster-listen")

var listenCmd = &cobra.Command{
	Use:     "listen",
	Aliases: []string{"l"},
	Short:   "Listen for events",
	Long:    `Listen for events`,
	RunE: func(cmd *cobra.Command, args []string) error {

		clientset, _, err := utils.MakeClientSet()
		if err != nil {
			return err
		}

		informer := cache.NewStore(clientset, logger.LevelDebug)
		informer.Start()

		go func() {
			informer.WatchEvery(1)
		}()

		utils.WaitStopAndExecute(func() {
			informer.Stop()
			stats := informer.GetStats()
			for podId, pendingDuration := range stats.PendingDurations {
				log.Info("Pod %s is pending for %d millisecond", podId.GetName(), pendingDuration.Milliseconds())
			}
			for podId, executionDuration := range stats.RunningDurations {
				log.Info("Pod %s is executed for %d millisecond", podId.GetName(), executionDuration.Milliseconds())
			}

			queueHistory := stats.GetPodQueueHistory()

			sort.Slice(queueHistory, func(i, j int) bool {
				return stats.PendingQHistory[i].At.Before(stats.PendingQHistory[j].At)
			})

			for _, sample := range queueHistory {
				log.Info("Pod queue length is %d at %s", sample.Value, sample.At.Format("2006-01-02 15:04:05.000"))
			}
		})
		return nil
	},
}
