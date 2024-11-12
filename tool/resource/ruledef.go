// Copyright (c) 2024 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package resource

import (
	"fmt"
	"strings"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/shared"
)

// -----------------------------------------------------------------------------
// Instrumentation Rule
//
// Instrumentation rules are used to define the behavior of the instrumentation
// for a specific function call. The rules are defined in the init() function
// of rule.go in each package directory. The rules are then used by the instrument
// package to generate the instrumentation code. Multiple rules can be defined
// for a single function call, and the rules are executed in the order of their
// priority. The rules are executed
// in the order of their priority, from high to low.
// There are several types of rules for different purposes:
// - InstFuncRule: Instrumentation rule for a specific function call
// - InstStructRule: Instrumentation rule for a specific struct type
// - InstFileRule: Instrumentation rule for a specific file

type InstRule interface {
	GetVersion() string    // GetVersion returns the version of the rule
	GetImportPath() string // GetImportPath returns import path of the rule
	GetPath() string       // GetPath returns the local path of the rule
	SetPath(path string)   // SetPath sets the local path of the rule
	String() string        // String returns string representation of rule
	Verify() error         // Verify checks the rule is valid
}

type InstBaseRule struct {
	Path       string // Local path of the rule
	Version    string // Version of the rule, e.g. "[1.9.1,1.9.2)" or ""
	ImportPath string // Import path of the rule, e.g. "github.com/gin-gonic/gin"
}

func (rule *InstBaseRule) GetVersion() string {
	return rule.Version
}

func (rule *InstBaseRule) GetImportPath() string {
	return rule.ImportPath
}

func (rule *InstBaseRule) GetPath() string {
	return rule.Path
}

func (rule *InstBaseRule) SetPath(path string) {
	rule.Path = path
}

// InstFuncRule finds specific function call and instrument by adding new code
type InstFuncRule struct {
	InstBaseRule
	Function     string // Function name, e.g. "New"
	ReceiverType string // Receiver type name, e.g. "*gin.Engine"
	Order        int    // Order of the rule, higher is executed first
	UseRaw       bool   // UseRaw indicates whether to insert raw code string
	OnEnter      string // OnEnter callback, called before original function
	OnExit       string // OnExit callback, called after original function
}

// InstStructRule finds specific struct type and instrument by adding new field
type InstStructRule struct {
	InstBaseRule
	StructType string // Struct type name, e.g. "Engine"
	FieldName  string // New field name, e.g. "Logger"
	FieldType  string // New field type, e.g. "zap.Logger"
}

// InstFileRule adds user file into compilation unit and do further compilation
type InstFileRule struct {
	InstBaseRule
	FileName string // File name, e.g. "engine.go"
	Replace  bool   // Replace indicates whether to replace the original file
}

func (rule *InstFuncRule) WithVersion(version string) *InstFuncRule {
	rule.Version = version
	return rule
}

func (rule *InstFuncRule) WithUseRaw(useRaw bool) *InstFuncRule {
	rule.UseRaw = useRaw
	return rule
}

func (rule *InstFuncRule) WithFileDeps(deps ...string) *InstFuncRule {
	return rule
}

func (rule *InstFileRule) WithReplace(replace bool) *InstFileRule {
	rule.Replace = replace
	return rule
}

func (rule *InstFileRule) WithVersion(version string) *InstFileRule {
	rule.Version = version
	return rule
}

// String returns string representation of the rule
func (rule *InstFuncRule) String() string {
	if rule.ReceiverType == "" {
		return fmt.Sprintf("%s@%s@%s {%s %s}",
			rule.ImportPath, rule.Version,
			rule.Function,
			rule.OnEnter, rule.OnExit)
	}
	return fmt.Sprintf("%s@%s@(%s).%s {%s %s}",
		rule.ImportPath, rule.Version,
		rule.ReceiverType, rule.Function,
		rule.OnEnter, rule.OnExit)
}

func (rule *InstStructRule) String() string {
	return fmt.Sprintf("%s@%s {%s}",
		rule.ImportPath, rule.Version,
		rule.StructType)
}

func (rule *InstFileRule) String() string {
	return fmt.Sprintf("%s@%s {%s}",
		rule.ImportPath, rule.Version,
		rule.FileName)
}

// Verify checks the rule is valid
func verifyRuleBase(rule *InstBaseRule) error {
	if rule.Path == "" {
		return fmt.Errorf("local path is empty")
	}
	if rule.ImportPath == "" {
		return fmt.Errorf("import path is empty")
	}
	if rule.Version != "" {
		// If version is specified, it should be in the format of [start,end)
		if !strings.Contains(rule.Version, "[") ||
			!strings.Contains(rule.Version, ")") ||
			!strings.Contains(rule.Version, ",") ||
			strings.Contains(rule.Version, "v") {
			return fmt.Errorf("invalid version format %s", rule.Version)
		}
	}
	return nil
}

func verifyRuleBaseWithoutPath(rule *InstBaseRule) error {
	if rule.ImportPath == "" {
		return fmt.Errorf("import path is empty")
	}
	if rule.Version != "" {
		// If version is specified, it should be in the format of [start,end)
		if !strings.Contains(rule.Version, "[") ||
			!strings.Contains(rule.Version, ")") ||
			!strings.Contains(rule.Version, ",") ||
			strings.Contains(rule.Version, "v") {
			return fmt.Errorf("invalid version format %s", rule.Version)
		}
	}
	return nil
}

func (rule *InstFileRule) Verify() error {
	err := verifyRuleBase(&rule.InstBaseRule)
	if err != nil {
		return err
	}
	if rule.FileName == "" {
		return fmt.Errorf("file name is empty")
	}
	if !shared.IsGoFile(rule.FileName) {
		return fmt.Errorf("file name should not end with .go")
	}
	return nil
}

func (rule *InstFuncRule) Verify() error {
	var err error
	if rule.UseRaw {
		err = verifyRuleBaseWithoutPath(&rule.InstBaseRule)
	} else {
		err = verifyRuleBase(&rule.InstBaseRule)
	}
	if err != nil {
		return err
	}
	if rule.Function == "" {
		return fmt.Errorf("function name is empty")
	}
	if rule.OnEnter == "" && rule.OnExit == "" {
		return fmt.Errorf("both onEnter and onExit are empty")
	}
	return nil
}

func (rule *InstStructRule) Verify() error {
	err := verifyRuleBaseWithoutPath(&rule.InstBaseRule)
	if err != nil {
		return err
	}
	if rule.StructType == "" {
		return fmt.Errorf("struct type is empty")
	}
	if rule.FieldName == "" || rule.FieldType == "" {
		return fmt.Errorf("field name is empty")
	}
	return nil
}