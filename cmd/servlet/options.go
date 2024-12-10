/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"k8c.io/servlet/internal/log"

	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

type Options struct {
	// NB: Not actually defined here, as ctrl-runtime registers its
	// own --kubeconfig flag that is required to make its GetConfigOrDie()
	// work.
	// KubeconfigFile string

	// PlatformKubeconfig is the kubeconfig that gives access
	// to the KDP platform cluster. This kubeconfig's cluster URL has to point to
	// the workspace where the KDP Service referenced via ServiceReference lives.
	PlatformKubeconfig string

	// Namespace is the namespace that the Servlet runs in.
	Namespace string

	// Whether or not to perform leader election (requires permissions to
	// manage coordination/v1 leases)
	EnableLeaderElection bool

	// ServletName can be used to give this Servlet a custom name. This name is used
	// for the Servlet resource inside the KDP platform. This value must not be changed
	// after a Servlet has registered for the first time in the platform.
	// If not given, defaults to "<service ref>-servlet".
	ServletName string

	// APIExportRef references the APIExport within a KDP organization workspace that this
	// servlet should work with by name. The APIExport has to already exist, but it must not have
	// pre-existing resource schemas configured.
	APIExportRef string

	PublishedResourceSelectorString string
	PublishedResourceSelector       labels.Selector

	LogOptions log.Options
}

func NewOptions() *Options {
	return &Options{
		LogOptions:                log.NewDefaultOptions(),
		PublishedResourceSelector: labels.Everything(),
	}
}

func (o *Options) AddFlags(flags *pflag.FlagSet) {
	o.LogOptions.AddPFlags(flags)

	flags.StringVar(&o.PlatformKubeconfig, "platform-kubeconfig", o.PlatformKubeconfig, "kubeconfig file of the KDP platform")
	flags.StringVar(&o.Namespace, "namespace", o.Namespace, "Kubernetes namespace the Servlet is running in")
	flags.StringVar(&o.ServletName, "servlet-name", o.ServletName, "name of this Servlet agent, must not be changed after the first run, can be left blank to auto-generate a name")
	flags.StringVar(&o.APIExportRef, "apiexport-ref", o.APIExportRef, "name of the APIExport in KDP that this Servlet is powering")
	flags.StringVar(&o.PublishedResourceSelectorString, "published-resource-selector", o.PublishedResourceSelectorString, "restrict this Servlet to only process PublishedResources matching this label selector (optional)")
	flags.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection, "whether to perform leader election")
}

func (o *Options) Validate() error {
	errs := []error{}

	if err := o.LogOptions.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(o.Namespace) == 0 {
		errs = append(errs, errors.New("--namespace is required"))
	}

	if len(o.ServletName) > 0 {
		if e := validation.IsDNS1035Label(o.ServletName); len(e) > 0 {
			errs = append(errs, fmt.Errorf("--servlet-name is invalid: %v", e))
		}
	}

	if len(o.APIExportRef) == 0 {
		errs = append(errs, errors.New("--apiexport-ref is required"))
	}

	if len(o.PlatformKubeconfig) == 0 {
		errs = append(errs, errors.New("--platform-kubeconfig is required"))
	}

	if s := o.PublishedResourceSelectorString; len(s) > 0 {
		if _, err := labels.Parse(s); err != nil {
			errs = append(errs, fmt.Errorf("invalid --published-resource-selector %q: %w", s, err))
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (o *Options) Complete() error {
	errs := []error{}

	if len(o.ServletName) == 0 {
		o.ServletName = o.APIExportRef + "-servlet"
	}

	if s := o.PublishedResourceSelectorString; len(s) > 0 {
		selector, err := labels.Parse(s)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid --published-resource-selector %q: %w", s, err))
		}
		o.PublishedResourceSelector = selector
	}

	return utilerrors.NewAggregate(errs)
}
