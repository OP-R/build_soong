// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/google/blueprint/proptools"
)

func init() {
	PreDepsMutators(func(ctx RegisterMutatorsContext) {
		ctx.BottomUp("variable", variableMutator).Parallel()
	})
}

type variableProperties struct {
	Product_variables struct {
		Platform_sdk_version struct {
			Asflags []string
		}

		// unbundled_build is a catch-all property to annotate modules that don't build in one or
		// more unbundled branches, usually due to dependencies missing from the manifest.
		Unbundled_build struct {
			Enabled *bool `android:"arch_variant"`
		} `android:"arch_variant"`

		Brillo struct {
			Cflags         []string
			Version_script *string `android:"arch_variant"`
		} `android:"arch_variant"`

		Malloc_not_svelte struct {
			Cflags []string
		}

		Safestack struct {
			Cflags []string `android:"arch_variant"`
		} `android:"arch_variant"`

		Cpusets struct {
			Cflags []string
		}

		Schedboost struct {
			Cflags []string
		}

		Binder32bit struct {
			Cflags []string
		}

		// debuggable is true for eng and userdebug builds, and can be used to turn on additional
		// debugging features that don't significantly impact runtime behavior.  userdebug builds
		// are used for dogfooding and performance testing, and should be as similar to user builds
		// as possible.
		Debuggable struct {
			Cflags   []string
			Cppflags []string
		}

		// eng is true for -eng builds, and can be used to turn on additionaly heavyweight debugging
		// features.
		Eng struct {
			Cflags   []string
			Cppflags []string
		}
	} `android:"arch_variant"`
}

var zeroProductVariables variableProperties

type productVariables struct {
	// Suffix to add to generated Makefiles
	Make_suffix *string `json:",omitempty"`

	Platform_sdk_version *int `json:",omitempty"`

	DeviceName        *string   `json:",omitempty"`
	DeviceArch        *string   `json:",omitempty"`
	DeviceArchVariant *string   `json:",omitempty"`
	DeviceCpuVariant  *string   `json:",omitempty"`
	DeviceAbi         *[]string `json:",omitempty"`
	DeviceUsesClang   *bool     `json:",omitempty"`
	DeviceVndkVersion *string   `json:",omitempty"`

	DeviceSecondaryArch        *string   `json:",omitempty"`
	DeviceSecondaryArchVariant *string   `json:",omitempty"`
	DeviceSecondaryCpuVariant  *string   `json:",omitempty"`
	DeviceSecondaryAbi         *[]string `json:",omitempty"`

	HostArch          *string `json:",omitempty"`
	HostSecondaryArch *string `json:",omitempty"`

	CrossHost              *string `json:",omitempty"`
	CrossHostArch          *string `json:",omitempty"`
	CrossHostSecondaryArch *string `json:",omitempty"`

	Allow_missing_dependencies *bool `json:",omitempty"`
	Unbundled_build            *bool `json:",omitempty"`
	Brillo                     *bool `json:",omitempty"`
	Malloc_not_svelte          *bool `json:",omitempty"`
	Safestack                  *bool `json:",omitempty"`
	HostStaticBinaries         *bool `json:",omitempty"`
	Cpusets                    *bool `json:",omitempty"`
	Schedboost                 *bool `json:",omitempty"`
	Binder32bit                *bool `json:",omitempty"`
	UseGoma                    *bool `json:",omitempty"`
	Debuggable                 *bool `json:",omitempty"`
	Eng                        *bool `json:",omitempty"`
	EnableCFI                  *bool `json:",omitempty"`

	VendorPath *string `json:",omitempty"`

	ClangTidy  *bool   `json:",omitempty"`
	TidyChecks *string `json:",omitempty"`

	DevicePrefer32BitExecutables *bool `json:",omitempty"`
	HostPrefer32BitExecutables   *bool `json:",omitempty"`

	SanitizeHost       []string `json:",omitempty"`
	SanitizeDevice     []string `json:",omitempty"`
	SanitizeDeviceArch []string `json:",omitempty"`

	ArtUseReadBarrier *bool `json:",omitempty"`

	BtConfigIncludeDir *string `json:",omitempty"`
	BtHcilpIncluded    *string `json:",omitempty"`
	BtHciUseMct        *bool   `json:",omitempty"`
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func (v *productVariables) SetDefaultConfig() {
	*v = productVariables{
		Platform_sdk_version:       intPtr(24),
		HostArch:                   stringPtr("x86_64"),
		HostSecondaryArch:          stringPtr("x86"),
		DeviceName:                 stringPtr("flounder"),
		DeviceArch:                 stringPtr("arm64"),
		DeviceArchVariant:          stringPtr("armv8-a"),
		DeviceCpuVariant:           stringPtr("denver64"),
		DeviceAbi:                  &[]string{"arm64-v8a"},
		DeviceUsesClang:            boolPtr(true),
		DeviceSecondaryArch:        stringPtr("arm"),
		DeviceSecondaryArchVariant: stringPtr("armv7-a-neon"),
		DeviceSecondaryCpuVariant:  stringPtr("denver"),
		DeviceSecondaryAbi:         &[]string{"armeabi-v7a"},
		Malloc_not_svelte:          boolPtr(false),
		Safestack:                  boolPtr(false),
	}

	if runtime.GOOS == "linux" {
		v.CrossHost = stringPtr("windows")
		v.CrossHostArch = stringPtr("x86")
		v.CrossHostSecondaryArch = stringPtr("x86_64")
	}
}

func variableMutator(mctx BottomUpMutatorContext) {
	var module Module
	var ok bool
	if module, ok = mctx.Module().(Module); !ok {
		return
	}

	// TODO: depend on config variable, create variants, propagate variants up tree
	a := module.base()
	variableValues := reflect.ValueOf(&a.variableProperties.Product_variables).Elem()
	zeroValues := reflect.ValueOf(zeroProductVariables.Product_variables)

	for i := 0; i < variableValues.NumField(); i++ {
		variableValue := variableValues.Field(i)
		zeroValue := zeroValues.Field(i)
		name := variableValues.Type().Field(i).Name
		property := "product_variables." + proptools.PropertyNameForField(name)

		// Check that the variable was set for the product
		val := reflect.ValueOf(mctx.Config().(Config).ProductVariables).FieldByName(name)
		if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
			continue
		}

		val = val.Elem()

		// For bools, check that the value is true
		if val.Kind() == reflect.Bool && val.Bool() == false {
			continue
		}

		// Check if any properties were set for the module
		if reflect.DeepEqual(variableValue.Interface(), zeroValue.Interface()) {
			continue
		}

		a.setVariableProperties(mctx, property, variableValue, val.Interface())
	}
}

func (a *ModuleBase) setVariableProperties(ctx BottomUpMutatorContext,
	prefix string, productVariablePropertyValue reflect.Value, variableValue interface{}) {

	printfIntoProperties(productVariablePropertyValue, variableValue)

	err := proptools.AppendMatchingProperties(a.generalProperties,
		productVariablePropertyValue.Addr().Interface(), nil)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			ctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			panic(err)
		}
	}
}

func printfIntoProperties(productVariablePropertyValue reflect.Value, variableValue interface{}) {
	for i := 0; i < productVariablePropertyValue.NumField(); i++ {
		propertyValue := productVariablePropertyValue.Field(i)
		kind := propertyValue.Kind()
		if kind == reflect.Ptr {
			if propertyValue.IsNil() {
				continue
			}
			propertyValue = propertyValue.Elem()
		}
		switch propertyValue.Kind() {
		case reflect.String:
			printfIntoProperty(propertyValue, variableValue)
		case reflect.Slice:
			for j := 0; j < propertyValue.Len(); j++ {
				printfIntoProperty(propertyValue.Index(j), variableValue)
			}
		case reflect.Bool:
			// Nothing
		case reflect.Struct:
			printfIntoProperties(propertyValue, variableValue)
		default:
			panic(fmt.Errorf("unsupported field kind %q", propertyValue.Kind()))
		}
	}
}

func printfIntoProperty(propertyValue reflect.Value, variableValue interface{}) {
	s := propertyValue.String()
	// For now, we only support int formats
	var i int
	if strings.Contains(s, "%d") {
		switch v := variableValue.(type) {
		case int:
			i = v
		case bool:
			if v {
				i = 1
			}
		default:
			panic(fmt.Errorf("unsupported type %T", variableValue))
		}
		propertyValue.Set(reflect.ValueOf(fmt.Sprintf(s, i)))
	}
}
