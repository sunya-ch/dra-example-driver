package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sigs.k8s.io/dra-example-driver/pkg/consts"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

var (
	commonAttributeCapacity = `
features:
  DRAConsumableCapacity: true
common:
  allowMultipleAllocations: true
  attributes:
    driverVersion:
      version: "1.0.0"
    model:
      string: "LATEST-GPU-MODEL"
  capacity:
    sm:
      value: "100"
      requestPolicy:
        default: "100"
        validRange:
          min: "10"
          step: "10"
          max: "100"
    memory:
      value: "80Gi"
      requestPolicy:
        default: "80Gi"
        validValues:
        - "5Gi"
        - "10Gi"
        - "20Gi"
        - "40Gi"
        - "80Gi"
`
	patch = `
features:
  DRAConsumableCapacity: true
patch:
  sharedCounters:
  - name: counter-set
    counters:
      memory:
        value: "80Gi"
  devices:
    gpu-1:
      attributes:
        dma:
          bool: true
`
)

func TestParseResourceSliceConfig(t *testing.T) {
	testCases := []struct {
		desc string
		data string
		want ResourceSliceConfig
		err  bool
	}{
		{
			desc: "empty",
			data: "",
			want: ResourceSliceConfig{
				Prefix: consts.DefaultPrefix,
			},
			err: false,
		},
		{
			desc: "common-attribute-capacity",
			data: commonAttributeCapacity,
			want: ResourceSliceConfig{
				Prefix: consts.DefaultPrefix,
				Common: Device{
					AllowMultipleAllocations: ptr.To(true),
					Attributes: map[string]DeviceAttribute{
						"model": {
							StringValue: ptr.To("LATEST-GPU-MODEL"),
						},
						"driverVersion": {
							VersionValue: ptr.To("1.0.0"),
						},
					},
					Capacity: map[string]DeviceCapacity{
						"sm": {
							Value: Quantity{Quantity: resource.MustParse("100")},
							RequestPolicy: &CapacityRequestPolicy{
								Default: &Quantity{Quantity: resource.MustParse("100")},
								ValidRange: &CapacityRequestPolicyRange{
									Min:  &Quantity{Quantity: resource.MustParse("10")},
									Step: &Quantity{Quantity: resource.MustParse("10")},
									Max:  &Quantity{Quantity: resource.MustParse("100")},
								},
							},
						},
						"memory": {
							Value: Quantity{Quantity: resource.MustParse("80Gi")},
							RequestPolicy: &CapacityRequestPolicy{
								Default: &Quantity{Quantity: resource.MustParse("80Gi")},
								ValidValues: []Quantity{
									{Quantity: resource.MustParse("5Gi")},
									{Quantity: resource.MustParse("10Gi")},
									{Quantity: resource.MustParse("20Gi")},
									{Quantity: resource.MustParse("40Gi")},
									{Quantity: resource.MustParse("80Gi")},
								},
							},
						},
					},
				},
			},
			err: false,
		},
		{
			desc: "patch",
			data: patch,
			want: ResourceSliceConfig{
				Prefix: consts.DefaultPrefix,
				Common: Device{},
				Patch: PatchConfig{
					SharedCounters: []CounterSet{
						{
							Name: "counter-set",
							Counters: map[string]Counter{
								"memory": {
									Value: Quantity{
										Quantity: resource.MustParse("80Gi"),
									},
								},
							},
						},
					},
					Devices: map[string]Device{
						"gpu-1": {
							Attributes: map[string]DeviceAttribute{
								"dma": {
									BoolValue: ptr.To(true),
								},
							},
						},
					},
				},
			},
			err: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fmt.Println(tc.data)
			result, err := parseResourceSliceConfig([]byte(tc.data))
			if !tc.err {
				require.NoError(t, err)
				assert.Equal(t, tc.want, result, "unexpected result")
			} else if err == nil {
				t.Fatal("expect error")
			}
		})
	}
}
