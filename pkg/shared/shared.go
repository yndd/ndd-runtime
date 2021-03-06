/*
Copyright 2022 NDD.

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
package shared

import (
	"time"

	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/registrator/registrator"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"github.com/yndd/ndd-runtime/pkg/targetchannel"
)

type NddControllerOptions struct {
	Logger                    logging.Logger
	Poll                      time.Duration
	Namespace                 string
	ControllerConfigName      string
	Revision                  string
	RevisionNamespace         string
	GnmiAddress               string
	CrdNames                  []string
	ServiceDiscoveryNamespace string
	Copts                     controller.Options
	Registrator               registrator.Registrator
	TargetCh                  chan targetchannel.TargetMsg
}
