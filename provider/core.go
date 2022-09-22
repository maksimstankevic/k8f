package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"k8f/core"
	"os"
	"strings"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// evaluate latest version from addon version list
func evaluateVersion(list []string) string {
	var latest string
	for _, v := range list {
		var lt *version.Version
		var err error
		v1, err := version.NewVersion(v)
		core.OnErrorFail(err, "Error Evaluating Version")
		if latest == "" {
			lt, err = version.NewVersion("0.0")
		} else {
			lt, err = version.NewVersion(latest)
		}
		core.OnErrorFail(err, "Error Evaluating Version")
		// Options availabe
		if v1.GreaterThan(lt) {
			latest = v
		} // GreaterThen
	}
	return latest
}

func RunResult(p interface{}, output string) string {
	var kJson []byte
	log.Debug("start RunResult Func")
	if output == "json" {
		log.Info("start Json Marshal")
		kJson, _ = json.Marshal(p)
	} else if output == "yaml" {
		log.Info("start YAML Marshal")
		kJson, _ = yaml.Marshal(p)
	} else {
		return "Requested Output is not supported"
	}
	log.Info("returning Output Marshal")
	log.Debug("returning Output Marshal")
	return string(kJson)
}

// func to count ammount of Cluster in an account
func countTotal(f []Account) int {
	var count int
	for _, a := range f {
		count = count + a.TotalCount
	}
	return count
}

// func to merge kubeconfig output to singe config file
func (c CommandOptions) Merge(configs AllConfig, arn string) {
	clientConfig := Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       configs.clusters,
		Contexts:       configs.context,
		CurrentContext: arn,
		Preferences:    Preferences{},
		Users:          configs.auth,
	}
	if c.DryRun {
		fmt.Println(RunResult(clientConfig, c.Output))
	} else {
		if c.Backup {
			log.Debug("calling copy file to bak")
			c.Configcopy()
		}
		y, _ := yaml.Marshal(clientConfig)
		err := os.WriteFile(c.Path, y, 0666)
		core.OnErrorFail(err, "failed to save config")

	}
}

func (c CommandOptions) FullCloudConfig() {
	var auth []Users
	var context []Contexts
	var clusters []Clusters
	r := make(chan AllConfig)
	for _, cloud := range []string{"azure", "aws"} {
		go func(cloud string, r chan AllConfig, c CommandOptions) {
			var res AllConfig
			if cloud == "azure" {
				res = c.ConnectAllAks()
			} else if cloud == "aws" {
				res = c.ConnectAllEks()
			}
			r <- res
		}(cloud, r, c)
	}
	for i := 0; i < len([]string{"azure", "aws"}); i++ {
		response := <-r
		auth = append(auth, response.auth...)
		context = append(context, response.context...)
		clusters = append(clusters, response.clusters...)
	}
	c.Merge(AllConfig{auth: auth, context: context, clusters: clusters}, context[0].Context.User)
}
func (c CommandOptions) Configcopy() {
	sourceFileStat, err := os.Stat(c.Path)
	core.OnErrorFail(err, "Issue Findign the Files in the path: "+c.Path)
	if !sourceFileStat.Mode().IsRegular() {
		core.OnErrorFail(err, c.Path+" is not a regular file")
	}
	source, err := os.Open(c.Path)
	core.OnErrorFail(err, "failed to Open target file")
	defer source.Close()

	destination, err := os.Create(c.Path + ".bak")
	core.OnErrorFail(err, "failed to Copy target file")
	defer destination.Close()
	_, err = io.Copy(destination, source)
	core.OnErrorFail(err, "failed to Copy target file")
}
func SplitAzIDAndGiveItem(input string, seperator string, out int) string {
	s := strings.Split(input, seperator)
	log.Debug("Split output")
	log.Debug(s)
	log.Debug(s[out])
	return s[out]
}
