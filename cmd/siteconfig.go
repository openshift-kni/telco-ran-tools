/*
Copyright 2023 Red Hat, Inc.

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

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/openshift-kni/telco-ran-tools/cmd/generated"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const SavePartlabel = "--save-partlabel"
const BootIgn = "boot-beauty.ign"
const DiscoveryIgn = "discovery-beauty.ign"
const TestBootIgn = "test-boot-beauty.ign"
const TestDiscoveryIgn = "test-discovery-beauty.ign"

// siteconfigCmd represents the siteconfig command.
var siteconfigCmd = &cobra.Command{
	Use:   "siteconfig",
	Short: "Update site config",
	Run: func(cmd *cobra.Command, args []string) {
		label, _ := cmd.Flags().GetString("label")
		cfgfile, _ := cmd.Flags().GetString("cfg")
		indent, _ := cmd.Flags().GetInt("indent")
		testMode, _ := cmd.Flags().GetBool("testmode")

		err := updateSiteConfig(label, cfgfile, indent, testMode)
		if err != nil {
			// Explicitly printing error here rather than returning err to avoid "usage" message in output
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
	Version: Version,
}

// nolint: errcheck
func init() {
	siteconfigCmd.Flags().StringP("label", "l", "data", "Partition label")
	siteconfigCmd.Flags().StringP("cfg", "c", "", "Site-config file")
	siteconfigCmd.Flags().IntP("indent", "i", 2, "Indentation")
	siteconfigCmd.Flags().Bool("testmode", false, "Use dummy ignition data for testing")

	rootCmd.AddCommand(siteconfigCmd)

}

// findYamlNode searches the node for a given key, but does not search sub-nodes.
func findYamlNode(root *yaml.Node, key string) *yaml.Node {
	for i, node := range root.Content {
		if node.Kind == yaml.ScalarNode && node.Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

// jqCompact uses the jq utility to compact a JSON string.
func jqCompact(s string) (string, error) {
	cmd := exec.Command("jq", "-cM")
	cmd.Stdin = strings.NewReader(s)
	out, err := cmd.Output()
	return strings.TrimSuffix(string(out), "\n"), err
}

// jqPruneIgnition uses the jq utility to prune prestaging data from the existing site-config that may be stale.
// Using the jq utility allows us to avoid unnecessary reordering of the JSON string.
func jqPruneIgnition(s string) (string, error) {
	jqCmd := `
		if type == "null" then empty else

		if .systemd?.units then
			# Delete prestaging service-units, including legacy content
			del( .systemd.units[]
				| select(
					.name == "precache-images.service"
					or .name == "precache-ocp-images.service"
					or ( .name == "var-mnt.mount" and (.contents | test ("BindsTo=precache")) )
					)
				)
		else . end

		| if .storage?.files then
			# Delete prestaging files, including legacy content
			del( .storage.files[]
				| select(
					.path == "/usr/local/bin/extract-ai.sh"
					or .path == "/usr/local/bin/extract-ocp.sh"
					or .path == "/usr/local/bin/agent-fix-bz1964591"
					)
				)
		else . end

		# Delete empty objects that remain
		| walk(
			if type == "object" then
				with_entries(if .value == {} or .value == [] then empty else . end)
			else .  end)

		# Delete entries where the only thing left is the "ignition" data
		| del(select(keys == ["ignition"]))

		| if type == "null" then empty else . end

		end
	`
	cmd := exec.Command("jq", "-cM", jqCmd)
	cmd.Stdin = strings.NewReader(s)
	out, err := cmd.Output()
	return strings.TrimSuffix(string(out), "\n"), err
}

// jqMergeIgnition uses the jq utility to merge prestaging data into the existing site-config.
// Using the jq utility allows us to avoid unnecessary reordering of the JSON string.
func jqMergeIgnition(s1, s2 string) (string, error) {
	jqCmd := `
		if length == 1 then
			# Only one element means there were no entries in the original
			# site-config (or all have been pruned), so just return the prestaging config
			.[0]
		else
			# Append the prestaging systemd units to the original site-config
			if .[0].systemd then
				if .[0].systemd.units then
					.[0].systemd.units += .[1].systemd.units
				else
					.[0].systemd.units = .[1].systemd.units
				end
			else
				.[0].systemd = .[1].systemd
			end
			# Append the prestaging storage files to the original site-config
			| if .[0].storage then
				if .[0].storage.files then
					.[0].storage.files += .[1].storage.files
				else
					.[0].storage.files = .[1].storage.files
				end
			else
				.[0].storage = .[1].storage
			end
			| .[0]
		end
	`
	cmd := exec.Command("jq", "-csM", jqCmd)
	cmd.Stdin = strings.NewReader(s1 + s2)
	out, err := cmd.Output()
	return strings.TrimSuffix(string(out), "\n"), err
}

// processIgnitionConfig handles either the cluster or node ignitionConfigOverride, merging prestaging data into it.
func processIgnitionConfig(rootPtr, ignitionPtr *yaml.Node, label, ignfile string) error {
	// Get the prestaging data to be merged
	prestagingIgnitionBytes, err := generated.Asset(ignfile)
	if err != nil {
		return err
	}

	prestagingIgnition, err := jqCompact(string(prestagingIgnitionBytes))
	if err != nil {
		return err
	}

	if label != "data" {
		// Swap the partition label in the prestagingIgnition
		prestagingIgnition = strings.ReplaceAll(prestagingIgnition, "--label data", fmt.Sprintf("--label %s", label))
	}

	if ignitionPtr == nil {
		// The site-config does not currently have ignitionConfigOverride at this position, so add it
		var key, value yaml.Node
		key.SetString("ignitionConfigOverride")
		value.SetString(prestagingIgnition)
		rootPtr.Content = append(rootPtr.Content, &key, &value)

		return nil
	}

	// Prune any prior prestaging ignition data, including older entries no longer in use
	sitecfgIgnition, err := jqPruneIgnition(ignitionPtr.Value)
	if err != nil {
		return err
	}

	// Merge the prestaging ignition data into the site-config
	ignitionPtr.Value, err = jqMergeIgnition(sitecfgIgnition, prestagingIgnition)
	if err != nil {
		return err
	}

	return nil
}

// processInstallerArgs handles the installerArgs, adding the --save-partlabel args if not already present.
func processInstallerArgs(nodePtr, nodeInstallerArgsPtr *yaml.Node, label string) error {
	requiredArgs := []string{SavePartlabel, label}

	if nodeInstallerArgsPtr == nil {
		// There isn't already an installerArgs value, so add one
		argsJson, err := json.Marshal(requiredArgs)
		if err != nil {
			return err
		}

		var key, value yaml.Node
		key.SetString("installerArgs")
		value.SetString(string(argsJson))
		nodePtr.Content = append(nodePtr.Content, &key, &value)

		return nil
	}

	// Get the current installerArgs and check whether the required args already exist
	var args []string
	err := json.Unmarshal([]byte(nodeInstallerArgsPtr.Value), &args)
	if err != nil {
		return err
	}

	for i, arg := range args {
		if arg == SavePartlabel {
			if len(args) > i && args[i+1] == label {
				// Nothing to do, already exists
				return nil
			}
		}
	}

	// Add the args
	args = append(args, requiredArgs...)
	argsJson, err := json.Marshal(args)
	if err != nil {
		return err
	}

	nodeInstallerArgsPtr.Value = string(argsJson)

	return nil
}

// readFileOrStdin will read site-config content from file or stdin.
func readFileOrStdin(cfgfile string) ([]byte, error) {
	if cfgfile == "" || cfgfile == "-" {
		// Read from stdin
		return io.ReadAll(os.Stdin)
	}

	return os.ReadFile(cfgfile)
}

// updateSiteConfig will update the existing site-config with prestaging data as needed.
func updateSiteConfig(label, cfgfile string, indent int, testMode bool) error {
	var bootIgn, discoveryIgn string
	if testMode {
		bootIgn = TestBootIgn
		discoveryIgn = TestDiscoveryIgn
	} else {
		bootIgn = BootIgn
		discoveryIgn = DiscoveryIgn
	}

	cfgbytes, err := readFileOrStdin(cfgfile)
	if err != nil {
		return err
	}

	if len(cfgbytes) == 0 {
		return fmt.Errorf("Invalid input: Empty file")
	}

	// Parse the site-config yaml into a yaml.Node structure and search for the required entry points
	var cfgdoc yaml.Node

	err = yaml.Unmarshal(cfgbytes, &cfgdoc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing filter file: %s\n", cfgfile)
		return err
	}

	specPtr := findYamlNode(cfgdoc.Content[0], "spec")
	if specPtr == nil {
		return fmt.Errorf("Failed to parse site-config: spec not found")
	}

	clusterSetPtr := findYamlNode(specPtr, "clusters")
	if clusterSetPtr == nil {
		return fmt.Errorf("Failed to parse site-config: clusters not found")
	}

	for _, clusterPtr := range clusterSetPtr.Content {
		// Update the cluster data, as needed
		clusterIgnitionPtr := findYamlNode(clusterPtr, "ignitionConfigOverride")
		err = processIgnitionConfig(clusterPtr, clusterIgnitionPtr, label, discoveryIgn)
		if err != nil {
			return err
		}

		nodeSetPtr := findYamlNode(clusterPtr, "nodes")
		if nodeSetPtr == nil {
			return fmt.Errorf("Failed to parse site-config: nodes not found")
		}

		for _, nodePtr := range nodeSetPtr.Content {
			// Update the node data, as needed
			nodeIgnitionPtr := findYamlNode(nodePtr, "ignitionConfigOverride")
			err = processIgnitionConfig(nodePtr, nodeIgnitionPtr, label, bootIgn)
			if err != nil {
				return err
			}

			nodeInstallerArgsPtr := findYamlNode(nodePtr, "installerArgs")
			err = processInstallerArgs(nodePtr, nodeInstallerArgsPtr, label)
			if err != nil {
				return err
			}
		}
	}

	// Display updated content
	var ybytes bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&ybytes)
	yamlEncoder.SetIndent(indent)

	err = yamlEncoder.Encode(&cfgdoc)
	if err != nil {
		return err
	}

	fmt.Print(ybytes.String())

	return nil
}
