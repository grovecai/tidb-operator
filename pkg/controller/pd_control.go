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

package controller

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
)

// getPDClientFromService gets the pd client from the TidbCluster
func getPDClientFromService(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster) pdapi.PDClient {
	if tc.Heterogeneous() && tc.WithoutLocalPD() {
		return pdControl.GetPDClient(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled(),
			pdapi.TLSCertFromTC(pdapi.Namespace(tc.GetNamespace()), tc.GetName()),
			pdapi.ClusterRef(tc.Spec.Cluster.ClusterDomain),
			pdapi.UseHeadlessService(tc.Spec.AcrossK8s),
		)
	}
	// cluster domain may be empty
	return pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled(), pdapi.ClusterRef(tc.Spec.ClusterDomain))
}

// getPDClientFromService gets the pd client from the TidbCluster
func getPDMSClientFromService(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster, serviceName string) pdapi.PDMSClient {
	if tc.Heterogeneous() && tc.WithoutLocalPD() {
		return pdControl.GetPDMSClient(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, serviceName, tc.IsTLSClusterEnabled(),
			pdapi.TLSCertFromTC(pdapi.Namespace(tc.GetNamespace()), tc.GetName()),
			pdapi.ClusterRef(tc.Spec.Cluster.ClusterDomain),
			pdapi.UseHeadlessService(tc.Spec.AcrossK8s),
		)
	}
	// cluster domain may be empty
	return pdControl.GetPDMSClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), serviceName,
		tc.IsTLSClusterEnabled(), pdapi.ClusterRef(tc.Spec.ClusterDomain))
}

// GetPDClient tries to return an available PDClient
// If the pdClient built from the PD service name is unavailable, try to
// build another one with the ClientURL in the PeerMembers.
// ClientURL example:
// ClientURL: https://cluster2-pd-0.cluster2-pd-peer.pingcap.svc.cluster2.local
func GetPDClient(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster) pdapi.PDClient {
	pdClient := getPDClientFromService(pdControl, tc)

	if len(tc.Status.PD.PeerMembers) == 0 {
		return pdClient
	}

	_, err := pdClient.GetHealth()
	if err == nil {
		return pdClient
	}

	for _, pdMember := range tc.Status.PD.PeerMembers {
		pdPeerClient := pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled(), pdapi.SpecifyClient(pdMember.ClientURL, pdMember.Name))
		_, err = pdPeerClient.GetHealth()
		if err == nil {
			return pdPeerClient
		}
	}

	return pdClient
}

// GetPDClientForMember tries to return a PDClient for a specific PD member.
func GetPDClientForMember(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster, member *v1alpha1.PDMember) pdapi.PDClient {
	if member == nil {
		return nil
	}
	return pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled(), pdapi.SpecifyClient(member.ClientURL, member.Name))
}

// GetPDMSClient tries to return an available PDMSClient
func GetPDMSClient(pdControl pdapi.PDControlInterface, tc *v1alpha1.TidbCluster, serviceName string) pdapi.PDMSClient {
	pdMSClient := getPDMSClientFromService(pdControl, tc, serviceName)

	err := pdMSClient.GetHealth()
	if err == nil {
		return pdMSClient
	}

	for _, service := range tc.Status.PDMS {
		if service.Name != serviceName {
			continue
		}
		for _, pdMember := range service.Members {
			pdMSPeerClient := pdControl.GetPDMSClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), serviceName,
				tc.IsTLSClusterEnabled(), pdapi.SpecifyClient(pdMember, pdMember))
			err = pdMSPeerClient.GetHealth()
			if err == nil {
				return pdMSPeerClient
			}
		}
	}

	return nil
}

// NewFakePDClient creates a fake pdclient that is set as the pd client
func NewFakePDClient(pdControl *pdapi.FakePDControl, tc *v1alpha1.TidbCluster) *pdapi.FakePDClient {
	pdClient := pdapi.NewFakePDClient()
	if tc.Spec.Cluster != nil {
		pdControl.SetPDClientWithClusterDomain(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.Spec.Cluster.ClusterDomain, pdClient)
	}
	if tc.Spec.ClusterDomain != "" {
		pdControl.SetPDClientWithClusterDomain(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.ClusterDomain, pdClient)
	}
	pdControl.SetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), pdClient)

	return pdClient
}

// NewFakePDClientForMember creates a fake pdclient that is set as the pd client for a specific PD member.
func NewFakePDClientForMember(pdControl *pdapi.FakePDControl, tc *v1alpha1.TidbCluster, member *v1alpha1.PDMember) *pdapi.FakePDClient {
	if member == nil {
		return nil
	}
	pdClient := pdapi.NewFakePDClient()
	pdControl.SetPDClientForKey(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), member.Name, pdClient)
	return pdClient
}

// NewFakePDMSClient creates a fake pdmsclient that is set as the pdms client
func NewFakePDMSClient(pdControl *pdapi.FakePDControl, tc *v1alpha1.TidbCluster, curService string) *pdapi.FakePDMSClient {
	pdmsClient := pdapi.NewFakePDMSClient()
	if tc.Spec.Cluster != nil {
		pdControl.SetPDMSClientWithClusterDomain(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.Spec.Cluster.ClusterDomain, curService, pdmsClient)
	}
	if tc.Spec.ClusterDomain != "" {
		pdControl.SetPDMSClientWithClusterDomain(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.ClusterDomain, curService, pdmsClient)
	}
	pdControl.SetPDMSClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), curService, pdmsClient)

	return pdmsClient
}

// NewFakePDClientWithAddress creates a fake pdclient that is set as the pd client
func NewFakePDClientWithAddress(pdControl *pdapi.FakePDControl, peerURL string) *pdapi.FakePDClient {
	pdClient := pdapi.NewFakePDClient()
	pdControl.SetPDClientWithAddress(peerURL, pdClient)
	return pdClient
}
