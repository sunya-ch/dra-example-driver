package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/dra-example-driver/pkg/consts"
)

var (
	ResourceSliceConfigPath = "/etc/config/slice-config.yaml"
)

type ResourceSliceConfig struct {
	DriverName           string          `yaml:"driverName,omitempty"`
	ExperimentalFeatures map[string]bool `yaml:"experimentalFeatures,omitempty"`
	Prefix               string          `yaml:"prefix,omitempty"`
	Common               Device          `yaml:"common,omitempty"`
	Patch                PatchConfig     `yaml:"patch,omitempty"`
}

type Device struct {
	AllowMultipleAllocations *bool                      `yaml:"allowMultipleAllocations,omitempty"`
	Attributes               map[string]DeviceAttribute `yaml:"attributes,omitempty"`
	Capacity                 map[string]DeviceCapacity  `yaml:"capacity,omitempty"`
}

type DeviceAttribute struct {
	IntValue     *int64  `yaml:"int,omitempty"`
	BoolValue    *bool   `yaml:"bool,omitempty"`
	StringValue  *string `yaml:"string,omitempty"`
	VersionValue *string `yaml:"version,omitempty"`
}

type DeviceCapacity struct {
	Value         Quantity               `yaml:"value,omitempty"`
	RequestPolicy *CapacityRequestPolicy `yaml:"requestPolicy,omitempty"`
}

func GetResourceSliceConfig() (ResourceSliceConfig, error) {
	data, err := os.ReadFile(ResourceSliceConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Gracefully skip
			klog.Infof("No ResourceSliceConfig %s found, use default prefix", ResourceSliceConfigPath)
			return ResourceSliceConfig{Prefix: consts.DefaultPrefix}, nil
		}
		return ResourceSliceConfig{}, err
	}
	return parseResourceSliceConfig(data)
}

func parseResourceSliceConfig(data []byte) (ResourceSliceConfig, error) {
	var cfg ResourceSliceConfig
	err := yaml.Unmarshal(data, &cfg)
	if err != nil {
		return ResourceSliceConfig{}, fmt.Errorf("failed to unmarshal data to ResourceSliceConfig: %v", err)
	}
	if cfg.Prefix == "" {
		cfg.Prefix = consts.DefaultPrefix
	}
	return cfg, nil
}

func (c ResourceSliceConfig) ConsumableCapacityFeature() bool {
	return c.ExperimentalFeatures["DRAConsumableCapacity"]
}

func (capacity DeviceCapacity) ConvertToResourceAPI() resourceapi.DeviceCapacity {
	apiCapacity := resourceapi.DeviceCapacity{
		Value: capacity.Value.Quantity,
	}
	if capacity.RequestPolicy != nil {
		apiCapacity.RequestPolicy = &resourceapi.CapacityRequestPolicy{}
		if capacity.RequestPolicy.Default != nil {
			apiCapacity.RequestPolicy.Default = &capacity.RequestPolicy.Default.Quantity
		}
		if capacity.RequestPolicy.ValidRange != nil {
			apiCapacity.RequestPolicy.ValidRange = capacity.RequestPolicy.ValidRange.ConvertToResourceAPI()
		}
		for _, value := range capacity.RequestPolicy.ValidValues {
			apiCapacity.RequestPolicy.ValidValues = append(apiCapacity.RequestPolicy.ValidValues, value.Quantity)
		}
	}
	return apiCapacity
}

type CapacityRequestPolicy struct {
	Default     *Quantity                   `yaml:"default,omitempty"`
	ValidValues []Quantity                  `yaml:"validValues,omitempty"`
	ValidRange  *CapacityRequestPolicyRange `yaml:"validRange,omitempty"`
}

type CapacityRequestPolicyRange struct {
	Min  *Quantity `yaml:"min,omitempty"`
	Max  *Quantity `yaml:"max,omitempty"`
	Step *Quantity `yaml:"step,omitempty"`
}

func (r *CapacityRequestPolicyRange) ConvertToResourceAPI() *resourceapi.CapacityRequestPolicyRange {
	validRange := &resourceapi.CapacityRequestPolicyRange{}
	if r.Min != nil {
		validRange.Min = &r.Min.Quantity
	}
	if r.Max != nil {
		validRange.Max = &r.Max.Quantity
	}
	if r.Step != nil {
		validRange.Step = &r.Step.Quantity
	}
	return validRange
}

type Quantity struct {
	resource.Quantity
}

func (q *Quantity) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		qty, err := resource.ParseQuantity(s)
		if err != nil {
			return err
		}
		q.Quantity = qty
		return nil
	}

	var i int64
	if err := value.Decode(&i); err == nil {
		q.Quantity = *resource.NewQuantity(i, resource.DecimalSI)
		return nil
	}

	return fmt.Errorf("cannot parse quantity: %v", value.Value)
}

type PatchConfig struct {
	SharedCounters []CounterSet      `yaml:"sharedCounters,omitempty"`
	Devices        map[string]Device `yaml:"devices,omitempty"`
}

type CounterSet struct {
	Name     string             `yaml:"name"`
	Counters map[string]Counter `yaml:"counters"`
}

func (cs *CounterSet) ConvertToResourceAPI() resourceapi.CounterSet {
	counterSet := resourceapi.CounterSet{
		Name: cs.Name,
	}
	if len(cs.Counters) > 0 {
		counterSet.Counters = make(map[string]resourceapi.Counter, 0)
	}
	for name, c := range cs.Counters {
		counterSet.Counters[name] = resourceapi.Counter{
			Value: c.Value.Quantity,
		}
	}
	return counterSet
}

type Counter struct {
	Value Quantity
}

func (d *DeviceAttribute) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]interface{}
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("DeviceAttribute must be a map: %v", err)
	}

	var setCount int

	for k, v := range raw {
		switch k {
		case "int":
			n, ok := v.(int)
			if !ok {
				// YAML numbers may come as int or float64
				f, ok := v.(float64)
				if !ok {
					return fmt.Errorf("int value must be a number, got %T", v)
				}
				n = int(f)
			}
			d.IntValue = new(int64)
			*d.IntValue = int64(n)
			setCount++
		case "bool":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("bool value must be a boolean, got %T", v)
			}
			d.BoolValue = new(bool)
			*d.BoolValue = b
			setCount++
		case "string":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("string value must be a string, got %T", v)
			}
			d.StringValue = new(string)
			*d.StringValue = s
			setCount++
		case "version":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("version value must be a string, got %T", v)
			}
			d.VersionValue = new(string)
			*d.VersionValue = s
			setCount++
		default:
			return fmt.Errorf("unknown field %q for DeviceAttribute", k)
		}
	}

	if setCount != 1 {
		return fmt.Errorf("DeviceAttribute must have exactly one field set, got %d", setCount)
	}

	return nil
}
