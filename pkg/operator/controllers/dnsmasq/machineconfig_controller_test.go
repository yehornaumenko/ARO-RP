package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestMachineConfigReconciler(t *testing.T) {
	fakeDh := func(controller *gomock.Controller) *mock_dynamichelper.MockInterface {
		return mock_dynamichelper.NewMockInterface(controller)
	}

	tests := []struct {
		name       string
		objects    []client.Object
		mocks      func(mdh *mock_dynamichelper.MockInterface)
		request    ctrl.Request
		wantErrMsg string
	}{
		{
			name:       "no cluster",
			objects:    []client.Object{},
			mocks:      func(mdh *mock_dynamichelper.MockInterface) {},
			request:    ctrl.Request{},
			wantErrMsg: "clusters.aro.openshift.io \"cluster\" not found",
		},
		{
			name: "controller disabled",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: "false",
						},
					},
				},
			},
			mocks:      func(mdh *mock_dynamichelper.MockInterface) {},
			request:    ctrl.Request{},
			wantErrMsg: "",
		},
		{
			name: "no MachineConfigPool for MachineConfig does nothing",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: "true",
						},
					},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg: "",
		},
		{
			name: "valid MachineConfigPool for MachineConfig reconciles MachineConfig",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: "true",
						},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.AssignableToTypeOf(&mcv1.MachineConfig{})).Times(1)
			},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			client := ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				Build()
			dh := fakeDh(controller)
			tt.mocks(dh)

			r := NewMachineConfigReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
				client,
				dh,
			)

			_, err := r.Reconcile(context.Background(), tt.request)

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
		})
	}
}
