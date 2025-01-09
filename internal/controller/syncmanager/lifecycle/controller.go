/*
Copyright 2025 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lifecycle

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// Controller is a controller-runtime controller
// that can be stopped by cancelling its root context.
type Controller struct {
	// a Controller representing the virtual workspace for the APIExport
	obj controller.Controller

	// a signal that is closed when the vwController has stopped
	stopped chan struct{}

	// a function that is used to stop the vwController
	cancelFunc context.CancelCauseFunc
}

func NewController(upstream controller.Controller) (Controller, error) {
	return Controller{
		obj: upstream,
	}, nil
}

// Start starts the wrapped controller.
func (c *Controller) Start(ctx context.Context, log *zap.SugaredLogger) error {
	if c.obj == nil {
		return errors.New("cannot restart a stopped controller")
	}

	if c.stopped != nil {
		return errors.New("controller is already running")
	}

	ctrlCtx, cancel := context.WithCancelCause(ctx)

	c.cancelFunc = cancel
	c.stopped = make(chan struct{})

	// start the controller in a new goroutine
	go func() {
		defer close(c.stopped)

		// this call blocks until ctrlCtx is done or an error occurs
		// like failing to start the watches
		if err := c.obj.Start(ctrlCtx); err != nil {
			log.Errorw("Controller has failed", zap.Error(err))
		}

		cancel(errors.New("closing to prevent leakage"))

		c.obj = nil
		c.cancelFunc = nil
	}()

	return nil
}

func (c *Controller) Running() bool {
	if c.obj == nil {
		return false
	}

	if c.stopped == nil {
		return false
	}

	select {
	case <-c.stopped:
		return false

	default:
		return true
	}
}

func (c *Controller) Stop(log *zap.SugaredLogger, cause error) error {
	if !c.Running() {
		return errors.New("controller is not running")
	}

	c.cancelFunc(cause)
	log.Info("Waiting for controller to shut down…")
	<-c.stopped
	log.Info("Controller has finished shutting down.")

	return nil
}
