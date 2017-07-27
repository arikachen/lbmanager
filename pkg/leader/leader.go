package leader

import (
	"fmt"
	"os"
	"time"

	clientv1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

const (
	hb = "leader"

	DefaultLeaseDuration = 10 * time.Second
	DefaultRenewDeadline = 5 * time.Second
	DefaultRetryPeriod   = 2 * time.Second
)

type IHandler interface {
	Init()
	Run(bool, <-chan struct{})
	Change()
}

func Run(obj interface{}) error {
	eventBroadcaster := record.NewBroadcaster()

	id, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("unable to get hostname: %v", err)
	}
	recorder := eventBroadcaster.NewRecorder(runtime.NewScheme(), clientv1.EventSource{
		Component: "leader-elector",
		Host:      id,
	})

	rl, err := New(hb,
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		return err
	}
	obj.(IHandler).Init()
	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: DefaultLeaseDuration,
		RenewDeadline: DefaultRenewDeadline,
		RetryPeriod:   DefaultRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(stop <-chan struct{}) {
				obj.(IHandler).Run(true, stop)
			},
			OnStoppedLeading: func() {
				obj.(IHandler).Change()
			},
			OnNewLeader: func(identity string) {
				if rl.Identity() != identity {
					obj.(IHandler).Change()
				}
			},
		},
	})
	return nil
}
