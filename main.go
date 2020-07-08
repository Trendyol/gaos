/*
Copyright 2020 The Gaos Authors.

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
	"zeus-gitlab.trendyol.com/general/gaos/cmd"
	"zeus-gitlab.trendyol.com/general/gaos/logger"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

var LOG = logger.NewLoggers()

func main() {
	cmd.Execute(version, builtBy, date, commit)
}
