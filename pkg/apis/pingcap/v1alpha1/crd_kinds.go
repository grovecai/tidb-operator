// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	Version   = "v1alpha1"
	GroupName = "pingcap.com"

	TiDBClusterName    = "tidbclusters"
	TiDBClusterKind    = "TidbCluster"
	TiDBClusterKindKey = "tidbcluster"

	DMClusterName    = "dmclusters"
	DMClusterKind    = "DMCluster"
	DMClusterKindKey = "dmcluster"

	BackupName    = "backups"
	BackupKind    = "Backup"
	BackupKindKey = "backup"

	RestoreName    = "restores"
	RestoreKind    = "Restore"
	RestoreKindKey = "restore"

	BackupScheduleName    = "backupschedules"
	BackupScheduleKind    = "BackupSchedule"
	BackupScheduleKindKey = "backupschedule"

	TiDBMonitorName    = "tidbmonitors"
	TiDBMonitorKind    = "TidbMonitor"
	TiDBMonitorKindKey = "tidbmonitor"

	TiDBInitializerName    = "tidbinitializers"
	TiDBInitializerKind    = "TidbInitializer"
	TiDBInitializerKindKey = "tidbinitializer"

	TiDBNGMonitoringName    = "tidbngmonitorings"
	TiDBNGMonitoringKind    = "TidbNGMonitoring"
	TiDBNGMonitoringKindKey = "tidbngmonitoring"

	TiDBDashboardName    = "tidbdashboards"
	TiDBDashboardKind    = "TidbDashboard"
	TiDBDashboardKindKey = "tidbdashboard"

	SpecPath = "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1."
)

type CrdKind struct {
	Kind                    string
	Plural                  string
	SpecName                string
	ShortNames              []string
	AdditionalPrinterColums []extensionsobj.CustomResourceColumnDefinition
}

type CrdKinds struct {
	KindsString      string
	TiDBCluster      CrdKind
	DMCluster        CrdKind
	Backup           CrdKind
	Restore          CrdKind
	BackupSchedule   CrdKind
	TiDBMonitor      CrdKind
	TiDBInitializer  CrdKind
	TiDBNGMonitoring CrdKind
}

var DefaultCrdKinds = CrdKinds{
	KindsString:      "",
	TiDBCluster:      CrdKind{Plural: TiDBClusterName, Kind: TiDBClusterKind, ShortNames: []string{"tc"}, SpecName: SpecPath + TiDBClusterKind},
	DMCluster:        CrdKind{Plural: DMClusterName, Kind: DMClusterKind, ShortNames: []string{"dc"}, SpecName: SpecPath + DMClusterKind},
	Backup:           CrdKind{Plural: BackupName, Kind: BackupKind, ShortNames: []string{"bk"}, SpecName: SpecPath + BackupKind},
	Restore:          CrdKind{Plural: RestoreName, Kind: RestoreKind, ShortNames: []string{"rt"}, SpecName: SpecPath + RestoreKind},
	BackupSchedule:   CrdKind{Plural: BackupScheduleName, Kind: BackupScheduleKind, ShortNames: []string{"bks"}, SpecName: SpecPath + BackupScheduleKind},
	TiDBMonitor:      CrdKind{Plural: TiDBMonitorName, Kind: TiDBMonitorKind, ShortNames: []string{"tm"}, SpecName: SpecPath + TiDBMonitorKind},
	TiDBInitializer:  CrdKind{Plural: TiDBInitializerName, Kind: TiDBInitializerKind, ShortNames: []string{"ti"}, SpecName: SpecPath + TiDBInitializerKind},
	TiDBNGMonitoring: CrdKind{Plural: TiDBNGMonitoringName, Kind: TiDBNGMonitoringKind, ShortNames: []string{"tngm"}, SpecName: SpecPath + TiDBNGMonitoringKind},
}
