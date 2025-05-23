// Copyright 2018 PingCAP, Inc.
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

package controller

import (
	"context"
	stderrs "errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
	fedv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// controllerKind contains the schema.GroupVersionKind for tidbcluster controller type.
	ControllerKind = v1alpha1.SchemeGroupVersion.WithKind("TidbCluster")

	// DMControllerKind contains the schema.GroupVersionKind for dmcluster controller type.
	DMControllerKind = v1alpha1.SchemeGroupVersion.WithKind("DMCluster")

	// BackupControllerKind contains the schema.GroupVersionKind for backup controller type.
	BackupControllerKind = v1alpha1.SchemeGroupVersion.WithKind("Backup")

	// CompactBackupControllerKind contains the schema.GroupVersionKind for backup controller type.
	CompactBackupControllerKind = v1alpha1.SchemeGroupVersion.WithKind("CompactBackup")

	// RestoreControllerKind contains the schema.GroupVersionKind for restore controller type.
	RestoreControllerKind = v1alpha1.SchemeGroupVersion.WithKind("Restore")

	// backupScheduleControllerKind contains the schema.GroupVersionKind for backupschedule controller type.
	backupScheduleControllerKind = v1alpha1.SchemeGroupVersion.WithKind("BackupSchedule")

	// tidbMonitorControllerKind contains the schema.GroupVersionKind for TidbMonitor controller type.
	tidbMonitorControllerKind = v1alpha1.SchemeGroupVersion.WithKind("TidbMonitor")

	// tidbNGMonitoringKind contains the schema.GroupVersionKind for TidbNGMonitoring controller type.
	tidbNGMonitoringKind = v1alpha1.SchemeGroupVersion.WithKind("TidbNGMonitoring")

	// tidbDashboardKind contains the schema.GroupVersionKind for TidbDashboard controller type.
	tidbDashboardKind = v1alpha1.SchemeGroupVersion.WithKind("TidbDashboard")

	// FedVolumeBackupControllerKind contains the schema.GroupVersionKind for federation VolumeBackup controller type.
	FedVolumeBackupControllerKind = fedv1alpha1.SchemeGroupVersion.WithKind("VolumeBackup")

	// FedVolumeRestoreControllerKind contains the schema.GroupVersionKind for federation VolumeRestore controller type.
	FedVolumeRestoreControllerKind = fedv1alpha1.SchemeGroupVersion.WithKind("VolumeRestore")

	// FedVolumeBackupScheduleControllerKind contains the schema.GroupVersionKind for federation VolumeBackupSchedule controller type.
	FedVolumeBackupScheduleControllerKind = fedv1alpha1.SchemeGroupVersion.WithKind("VolumeBackupSchedule")
)

// RequeueError is used to requeue the item, this error type should't be considered as a real error
type RequeueError struct {
	s string
}

func (re *RequeueError) Error() string {
	return re.s
}

// RequeueErrorf returns a RequeueError
func RequeueErrorf(format string, a ...interface{}) error {
	return &RequeueError{fmt.Sprintf(format, a...)}
}

// IsRequeueError returns whether err is a RequeueError
func IsRequeueError(err error) bool {
	rerr := &RequeueError{}
	return stderrs.As(err, &rerr)
}

// IgnoreError is used to ignore this item, this error type shouldn't be considered as a real error, no need to requeue
type IgnoreError struct {
	s string
}

func (re *IgnoreError) Error() string {
	return re.s
}

// IgnoreErrorf returns a IgnoreError
func IgnoreErrorf(format string, a ...interface{}) error {
	return &IgnoreError{fmt.Sprintf(format, a...)}
}

// IsIgnoreError returns whether err is a IgnoreError
func IsIgnoreError(err error) bool {
	_, ok := err.(*IgnoreError)
	return ok
}

// GetOwnerRef returns TidbCluster's OwnerReference
func GetOwnerRef(tc *v1alpha1.TidbCluster) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         ControllerKind.GroupVersion().String(),
		Kind:               ControllerKind.Kind,
		Name:               tc.GetName(),
		UID:                tc.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetDMOwnerRef returns DMCluster's OwnerReference
func GetDMOwnerRef(dc *v1alpha1.DMCluster) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         DMControllerKind.GroupVersion().String(),
		Kind:               DMControllerKind.Kind,
		Name:               dc.GetName(),
		UID:                dc.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetBackupOwnerRef returns Backup's OwnerReference
func GetBackupOwnerRef(backup *v1alpha1.Backup) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         BackupControllerKind.GroupVersion().String(),
		Kind:               BackupControllerKind.Kind,
		Name:               backup.GetName(),
		UID:                backup.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetCompactBackupOwnerRef returns Backup's OwnerReference
func GetCompactBackupOwnerRef(backup *v1alpha1.CompactBackup) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         CompactBackupControllerKind.GroupVersion().String(),
		Kind:               CompactBackupControllerKind.Kind,
		Name:               backup.GetName(),
		UID:                backup.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetRestoreOwnerRef returns Restore's OwnerReference
func GetRestoreOwnerRef(restore *v1alpha1.Restore) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         RestoreControllerKind.GroupVersion().String(),
		Kind:               RestoreControllerKind.Kind,
		Name:               restore.GetName(),
		UID:                restore.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetBackupScheduleOwnerRef returns BackupSchedule's OwnerReference
func GetBackupScheduleOwnerRef(bs *v1alpha1.BackupSchedule) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         backupScheduleControllerKind.GroupVersion().String(),
		Kind:               backupScheduleControllerKind.Kind,
		Name:               bs.GetName(),
		UID:                bs.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetFedVolumeBackupScheduleOwnerRef returns FedVolumeBackupSchedule's OwnerReference
func GetFedVolumeBackupScheduleOwnerRef(vbks *fedv1alpha1.VolumeBackupSchedule) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         FedVolumeBackupScheduleControllerKind.GroupVersion().String(),
		Kind:               FedVolumeBackupScheduleControllerKind.Kind,
		Name:               vbks.GetName(),
		UID:                vbks.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

func GetTiDBMonitorOwnerRef(monitor *v1alpha1.TidbMonitor) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         tidbMonitorControllerKind.GroupVersion().String(),
		Kind:               tidbMonitorControllerKind.Kind,
		Name:               monitor.GetName(),
		UID:                monitor.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

func GetTiDBNGMonitoringOwnerRef(tngm *v1alpha1.TidbNGMonitoring) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         tidbNGMonitoringKind.GroupVersion().String(),
		Kind:               tidbNGMonitoringKind.Kind,
		Name:               tngm.GetName(),
		UID:                tngm.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

func GetTiDBDashboardOwnerRef(td *v1alpha1.TidbDashboard) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         tidbDashboardKind.GroupVersion().String(),
		Kind:               tidbDashboardKind.Kind,
		Name:               td.GetName(),
		UID:                td.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// GetServiceType returns member's service type
func GetServiceType(services []v1alpha1.Service, serviceName string) corev1.ServiceType {
	for _, svc := range services {
		if svc.Name == serviceName {
			switch svc.Type {
			case "NodePort":
				return corev1.ServiceTypeNodePort
			case "LoadBalancer":
				return corev1.ServiceTypeLoadBalancer
			default:
				return corev1.ServiceTypeClusterIP
			}
		}
	}
	return corev1.ServiceTypeClusterIP
}

// TiKVCapacity returns string resource requirement. In tikv-server, KB/MB/GB
// equal to MiB/GiB/TiB, so we cannot use resource.String() directly.
// Minimum unit we use is MiB, capacity less than 1MiB is ignored.
// https://github.com/tikv/tikv/blob/v3.0.3/components/tikv_util/src/config.rs#L155-L168
// For backward compatibility with old TiKV versions, we should use GB/MB
// rather than GiB/MiB, see https://github.com/tikv/tikv/blob/v2.1.16/src/util/config.rs#L359.
func TiKVCapacity(limits corev1.ResourceList) string {
	defaultArgs := "0"
	if limits == nil {
		return defaultArgs
	}
	q, ok := limits[corev1.ResourceStorage]
	if !ok {
		return defaultArgs
	}
	i, b := q.AsInt64()
	if !b {
		klog.Errorf("quantity %s can't be converted to int64", q.String())
		return defaultArgs
	}
	if i%humanize.GiByte == 0 {
		return fmt.Sprintf("%dGB", i/humanize.GiByte)
	}
	return fmt.Sprintf("%dMB", i/humanize.MiByte)
}

// MemberName return a component member name
func MemberName(clusterName string, member v1alpha1.MemberType) string {
	return fmt.Sprintf("%s-%s", clusterName, member)
}

// PDMemberName returns pd member name
func PDMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pd", clusterName)
}

// PDPeerMemberName returns pd peer service name
func PDPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pd-peer", clusterName)
}

// PDMSMemberName returns pd microservice member name
func PDMSMemberName(clusterName string, serviceName string) string {
	return fmt.Sprintf("%s-%s", clusterName, serviceName)
}

// PDMSPeerMemberName returns pd microservice peer service name
func PDMSPeerMemberName(clusterName string, serviceName string) string {
	return fmt.Sprintf("%s-%s-peer", clusterName, serviceName)
}

// PDMSTrimName returns last `-` separated string for `PDMSMemberName`
func PDMSTrimName(memberName string) string {
	name := memberName[strings.LastIndex(memberName, "-")+1:]
	if name == "peer" {
		// get middle serviceName
		check := memberName[:strings.LastIndex(memberName, "-")]
		name = check[strings.LastIndex(check, "-")+1:]
	}
	return name
}

// TiKVMemberName returns tikv member name
func TiKVMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tikv", clusterName)
}

// TiKVPeerMemberName returns tikv peer service name
func TiKVPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tikv-peer", clusterName)
}

// TiFlashMemberName returns tiflash member name
func TiFlashMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflash", clusterName)
}

// TiFlashPeerMemberName returns tiflash peer service name
func TiFlashPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiflash-peer", clusterName)
}

// TiProxyMemberName returns tiproxy member name
func TiProxyMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiproxy", clusterName)
}

// TiProxyPeerMemberName returns tiproxy peer service name
func TiProxyPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tiproxy-peer", clusterName)
}

// TiCDCMemberName returns ticdc member name
func TiCDCMemberName(clusterName string) string {
	return fmt.Sprintf("%s-ticdc", clusterName)
}

// TiCDCPeerMemberName returns ticdc peer service name
func TiCDCPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-ticdc-peer", clusterName)
}

// TiDBMemberName returns tidb member name
func TiDBMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tidb", clusterName)
}

// TiDBPeerMemberName returns tidb peer service name
func TiDBPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tidb-peer", clusterName)
}

// PumpMemberName returns pump member name
func PumpMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pump", clusterName)
}

// TiDBInitializerMemberName returns TiDBInitializer member name
func TiDBInitializerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-tidb-initializer", clusterName)
}

// For backward compatibility, pump peer member name do not has -peer suffix
// PumpPeerMemberName returns pump peer service name
func PumpPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-pump", clusterName)
}

// DiscoveryMemberName returns the name of tidb discovery
func DiscoveryMemberName(clusterName string) string {
	return fmt.Sprintf("%s-discovery", clusterName)
}

// DMMasterMemberName returns dm-master member name
func DMMasterMemberName(clusterName string) string {
	return fmt.Sprintf("%s-dm-master", clusterName)
}

// DMMasterPeerMemberName returns dm-master peer service name
func DMMasterPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-dm-master-peer", clusterName)
}

// DMWorkerMemberName returns dm-worker member name
func DMWorkerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-dm-worker", clusterName)
}

// DMWorkerPeerMemberName returns dm-worker peer service name
func DMWorkerPeerMemberName(clusterName string) string {
	return fmt.Sprintf("%s-dm-worker-peer", clusterName)
}

// TiDBInitSecret returns tidb init secret name
func TiDBInitSecret(clusterName string) string {
	return fmt.Sprintf("%s-init", clusterName)
}

// AnnProm adds annotations for prometheus scraping metrics
func AnnProm(port int32, path string) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   path,
		"prometheus.io/port":   fmt.Sprintf("%d", port),
	}
}
func FormatClusterDomainForRegex(clusterDomain string) string {
	if clusterDomain == "" {
		return ""
	}
	return "(|" + regexp.QuoteMeta("."+clusterDomain) + ")"
}

func FormatClusterDomain(clusterDomain string) string {
	if clusterDomain == "" {
		return ""
	}
	return "." + clusterDomain
}

func PDPeerFullyDomain(name, ns, clusterDomain string) string {
	return fmt.Sprintf("%s.%s.svc%s", PDPeerMemberName(name), ns, FormatClusterDomain(clusterDomain))
}

// AnnAdditionalProm adds additional prometheus scarping configuration annotation for the pod
// which has multiple metrics endpoint
// we assumes that the metrics path is as same as the previous metrics path
func AnnAdditionalProm(name string, port int32) map[string]string {
	return map[string]string{
		fmt.Sprintf("%s.prometheus.io/port", name): fmt.Sprintf("%d", port),
	}
}

func ParseStorageRequest(req corev1.ResourceList) (corev1.ResourceRequirements, error) {
	if req == nil {
		return corev1.ResourceRequirements{}, nil
	}
	q, ok := req[corev1.ResourceStorage]
	if !ok {
		return corev1.ResourceRequirements{}, fmt.Errorf("storage request is not set")
	}
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: q,
		},
	}, nil
}

func ContainerResource(req corev1.ResourceRequirements) corev1.ResourceRequirements {
	trimmed := req.DeepCopy()
	if trimmed.Limits != nil {
		delete(trimmed.Limits, corev1.ResourceStorage)
	}
	if trimmed.Requests != nil {
		delete(trimmed.Requests, corev1.ResourceStorage)
	}
	return *trimmed
}

// MemberConfigMapName returns the default ConfigMap name of the specified member type
// Deprecated
// TODO: remove after helm get totally abandoned
func MemberConfigMapName(tc *v1alpha1.TidbCluster, member v1alpha1.MemberType) string {
	nameKey := fmt.Sprintf("%s-%s", tc.Name, member)
	return nameKey + getConfigMapSuffix(tc, member.String(), nameKey)
}

// getConfigMapSuffix return the ConfigMap name suffix
func getConfigMapSuffix(tc *v1alpha1.TidbCluster, component string, name string) string {
	if tc.Annotations == nil {
		return ""
	}
	sha := tc.Annotations[fmt.Sprintf("pingcap.com/%s.%s.sha", component, name)]
	if len(sha) == 0 {
		return ""
	}
	return "-" + sha
}

// setIfNotEmpty set the value into map when value in not empty
func setIfNotEmpty(container map[string]string, key, value string) {
	if value != "" {
		container[key] = value
	}
}

// RequestTracker is used by unit test for mocking request error
type RequestTracker struct {
	requests int
	err      error
	after    int
}

func (rt *RequestTracker) ErrorReady() bool {
	return rt.err != nil && rt.requests >= rt.after
}

func (rt *RequestTracker) Inc() {
	rt.requests++
}

func (rt *RequestTracker) Reset() {
	rt.err = nil
	rt.after = 0
}

func (rt *RequestTracker) SetError(err error) *RequestTracker {
	rt.err = err
	return rt
}

func (rt *RequestTracker) SetAfter(after int) *RequestTracker {
	rt.after = after
	return rt
}

func (rt *RequestTracker) SetRequests(requests int) *RequestTracker {
	rt.requests = requests
	return rt
}

func (rt *RequestTracker) GetRequests() int {
	return rt.requests
}

func (rt *RequestTracker) GetError() error {
	return rt.err
}

// WacthForObject watch the object change from informer and add it to workqueue
func WatchForObject(informer cache.SharedIndexInformer, q workqueue.Interface) {
	enqueueFn := func(obj interface{}) {
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
			return
		}
		q.Add(key)
	}
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: enqueueFn,
		UpdateFunc: func(_, cur interface{}) {
			enqueueFn(cur)
		},
		DeleteFunc: enqueueFn,
	})
}

type GetControllerFn func(ns, name string) (runtime.Object, error)

// WatchForController watch the object change from informer and add it's controller to workqueue
func WatchForController(informer cache.SharedIndexInformer, q workqueue.Interface, fn GetControllerFn, m map[string]string) {
	enqueueFn := func(obj interface{}) {
		meta, ok := obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("%+v is not a runtime.Object, cannot get controller from it", obj))
			return
		}
		if m != nil {
			l := meta.GetLabels()
			if !util.IsSubMapOf(m, l) {
				return
			}
		}
		ref := metav1.GetControllerOf(meta)
		if ref == nil {
			return
		}
		refGV, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("cannot parse group version for the controller %v of %s/%s",
				ref, meta.GetNamespace(), meta.GetName()))
			return
		}
		controllerObj, err := fn(meta.GetNamespace(), ref.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				klog.V(4).Infof("controller %s/%s of %s not found, ignore", meta.GetNamespace(), ref.Name, meta.GetName())
			} else {
				utilruntime.HandleError(fmt.Errorf("cannot get controller %s/%s of %s", meta.GetNamespace(), ref.Name, meta.GetName()))
			}
			return
		}
		// Ensure the ref is exactly the controller we listed
		if ref.Kind == controllerObj.GetObjectKind().GroupVersionKind().Kind &&
			refGV.Group == controllerObj.GetObjectKind().GroupVersionKind().Group {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(controllerObj)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", controllerObj, err))
				return
			}
			q.Add(key)
		}
	}
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: enqueueFn,
		UpdateFunc: func(_, cur interface{}) {
			enqueueFn(cur)
		},
		DeleteFunc: enqueueFn,
	})
}

// EmptyClone create an clone of the resource with the same name and namespace (if namespace-scoped), with other fields unset
func EmptyClone(obj client.Object) (client.Object, error) {
	meta, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("Obj %v is not a metav1.Object, cannot call EmptyClone", obj)
	}
	gvk, err := InferObjectKind(obj)
	if err != nil {
		return nil, err
	}
	inst, err := scheme.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	instMeta, ok := inst.(client.Object)
	if !ok {
		return nil, fmt.Errorf("New instatnce %v created from scheme is not a metav1.Object, EmptyClone failed", inst)
	}
	instMeta.SetName(meta.GetName())
	instMeta.SetNamespace(meta.GetNamespace())
	return instMeta, nil
}

func DeepCopyClientObject(input client.Object) client.Object {
	robj := input.DeepCopyObject()
	cobj := robj.(client.Object)
	return cobj
}

// InferObjectKind infers the object kind
func InferObjectKind(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if len(gvks) != 1 {
		return schema.GroupVersionKind{}, fmt.Errorf("object %v has ambiguous GVK", obj)
	}
	return gvks[0], nil
}

// GuaranteedUpdate will retry the updateFunc to mutate the object until success, updateFunc is expected to
// capture the object reference from the caller context to avoid unnecessary type casting.
func GuaranteedUpdate(cli client.Client, obj client.Object, updateFunc func() error) error {
	key := client.ObjectKeyFromObject(obj)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := cli.Get(context.TODO(), key, obj); err != nil {
			return err
		}
		beforeMutation := obj.DeepCopyObject()
		if err := updateFunc(); err != nil {
			return err
		}
		if apiequality.Semantic.DeepEqual(obj, beforeMutation) {
			return nil
		}
		return cli.Update(context.TODO(), obj)
	})
}
