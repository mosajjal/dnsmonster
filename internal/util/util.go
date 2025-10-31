/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

// Package util provides the general configuration and variable types needed for differnet parts of dnsmonster
// Logging, metrics, and the search trees for allowlist and skiplist are generated and updated here.
package util

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/golang-collections/collections/tst"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

var (
	globalMetricConfig metricConfig
	// GlobalParser is the top-level argument parser. each output, capture, metric etc flag is registered
	// under Globalparser. This makes it easier for output modules to incorporate their own flags
	GlobalParser = flags.NewNamedParser("dnsmonster", flags.PassDoubleDash|flags.PrintErrors)
	// GeneralFlags is an ad-hoc solution to make all the flags available
	// to capture, metrics, util and output plugins.
	GeneralFlags generalConfig
	GlobalCancel context.CancelFunc
)

type generalConfig struct {
	Config                      flags.Filename `long:"config"                      ini-name:"config"                      env:"DNSMONSTER_CONFIG"                      default:""                                                        no-ini:"true"                                                                                                            description:"path to config file"`
	GcTime                      time.Duration  `long:"gctime"                      ini-name:"gctime"                      env:"DNSMONSTER_GCTIME"                      default:"10s"                                                     description:"Garbage Collection interval for tcp assembly and ip defragmentation"`
	CaptureStatsDelay           time.Duration  `long:"capturestatsdelay"           ini-name:"capturestatsdelay"           env:"DNSMONSTER_CAPTURESTATSDELAY"           default:"1s"                                                      description:"Duration to calculate interface stats"`
	MaskSize4                   int            `long:"masksize4"                   ini-name:"masksize4"                   env:"DNSMONSTER_MASKSIZE4"                   default:"32"                                                      description:"Mask IPv4s by bits. 32 means all the bits of IP is saved in DB"`
	MaskSize6                   int            `long:"masksize6"                   ini-name:"masksize6"                   env:"DNSMONSTER_MASKSIZE6"                   default:"128"                                                     description:"Mask IPv6s by bits. 32 means all the bits of IP is saved in DB"`
	ServerName                  string         `long:"servername"                  ini-name:"servername"                  env:"DNSMONSTER_SERVERNAME"                  default:"default"                                                 description:"Name of the server used to index the metrics."`
	LogFormat                   string         `long:"logformat"                   ini-name:"logformat"                   env:"DNSMONSTER_LOGFORMAT"                   default:"text"                                                    description:"Set debug Log format"                                                                                       choice:"json" choice:"text"`
	LogLevel                    uint           `long:"loglevel"                    ini-name:"loglevel"                    env:"DNSMONSTER_LOGLEVEL"                    default:"3"                                                       description:"Set debug Log level, 0:PANIC, 1:ERROR, 2:WARN, 3:INFO, 4:DEBUG"                                             choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ResultChannelSize           uint           `long:"resultchannelsize"           ini-name:"resultchannelsize"           env:"DNSMONSTER_RESULTCHANNELSIZE"           default:"100000"                                                  description:"Size of the result processor channel size"`
	Cpuprofile                  string         `long:"cpuprofile"                  ini-name:"cpuprofile"                  env:"DNSMONSTER_CPUPROFILE"                  default:""                                                        description:"write cpu profile to file"`
	Memprofile                  string         `long:"memprofile"                  ini-name:"memprofile"                  env:"DNSMONSTER_MEMPROFILE"                  default:""                                                        description:"write memory profile to file"`
	Gomaxprocs                  int            `long:"gomaxprocs"                  ini-name:"gomaxprocs"                  env:"DNSMONSTER_GOMAXPROCS"                  default:"-1"                                                      description:"GOMAXPROCS variable"`
	PacketLimit                 int            `long:"packetlimit"                 ini-name:"packetlimit"                 env:"DNSMONSTER_PACKETLIMIT"                 default:"0"                                                       description:"Limit of packets logged to clickhouse every iteration. Default 0 (disabled)"`
	SkipDomainsFile             string         `long:"skipdomainsfile"             ini-name:"skipdomainsfile"             env:"DNSMONSTER_SKIPDOMAINSFILE"             default:""                                                        description:"Skip outputing domains matching items in the CSV file path. Can accept a URL (http:// or https://) or path"`
	SkipDomainsRefreshInterval  time.Duration  `long:"skipdomainsrefreshinterval"  ini-name:"skipdomainsrefreshinterval"  env:"DNSMONSTER_SKIPDOMAINSREFRESHINTERVAL"  default:"60s"                                                     description:"Hot-Reload skipdomainsfile interval"`
	SkipDomainsFileType         string         `long:"skipdomainsfiletype"         ini-name:"skipdomainsfiletype"         env:"DNSMONSTER_SKIPDOMAINSFILETYPE"         default:""                                                        hidden:"true"`
	AllowDomainsFile            string         `long:"allowdomainsfile"            ini-name:"allowdomainsfile"            env:"DNSMONSTER_ALLOWDOMAINSFILE"            default:""                                                        description:"Allow Domains logic input file. Can accept a URL (http:// or https://) or path"`
	AllowDomainsRefreshInterval time.Duration  `long:"allowdomainsrefreshinterval" ini-name:"allowdomainsrefreshinterval" env:"DNSMONSTER_ALLOWDOMAINSREFRESHINTERVAL" default:"60s"                                                     description:"Hot-Reload allowdomainsfile file interval"`
	AllowDomainsFileType        string         `long:"allowdomainsfiletype"        ini-name:"allowdomainsfiletype"        env:"DNSMONSTER_ALLOWDOMAINSFILETYPE"        default:""                                                        hidden:"true"`
	SkipTLSVerification         bool           `long:"skiptlsverification"         ini-name:"skiptlsverification"         env:"DNSMONSTER_SKIPTLSVERIFICATION"         description:"Skip TLS verification when making HTTPS connections"`
	Version                     bool           `long:"version"                     ini-name:"version"                     env:"DNSMONSTER_VERSION"                     description:"show version and quit."                              no-ini:"true"`
	// used to implement allowdomains logic
	allowPrefixTst *tst.TernarySearchTree
	allowSuffixTst *tst.TernarySearchTree
	allowTypeHt    map[string]uint8
	// used to implement skipdomains logic
	skipPrefixTst *tst.TernarySearchTree
	skipSuffixTst *tst.TernarySearchTree
	skipTypeHt    map[string]uint8
}

func (g generalConfig) LoadAllowDomain() {
	GeneralFlags.allowPrefixTst, GeneralFlags.allowSuffixTst, GeneralFlags.allowTypeHt = LoadDomainsCsv(GeneralFlags.AllowDomainsFile)
}

func (g generalConfig) LoadSkipDomain() {
	GeneralFlags.skipPrefixTst, GeneralFlags.skipSuffixTst, GeneralFlags.skipTypeHt = LoadDomainsCsv(GeneralFlags.SkipDomainsFile)
}

var helpOptions struct {
	Help           bool           `long:"help"           ini-name:"help" short:"h" no-ini:"true" description:"Print this help to stdout"`
	ManPage        bool           `long:"manpage"        ini-name:"manpage"        no-ini:"true" description:"Print Manpage for dnsmonster to stdout"`
	BashCompletion bool           `long:"bashcompletion" ini-name:"bashcompletion" no-ini:"true" description:"Print bash completion script to stdout"`
	FishCompletion bool           `long:"fishcompletion" ini-name:"fishcompletion" no-ini:"true" description:"Print fish completion script to stdout"`
	SystemdService bool           `long:"systemdservice" ini-name:"systemdservice" no-ini:"true" description:"Print a sample systemd service to stdout"`
	WriteConfig    flags.Filename `long:"writeconfig"    ini-name:"writeconfig"    no-ini:"true" description:"generate a config file based on current inputs and write to provided path" default:""`
}

// GetCommitHash retrieves the current commit hash from the build information.
func GetCommitHash() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return "unknown" // Or handle the case where info isn't available
}

// GetCommitDate retrieves the current commit date from the build information.
func GetCommitDate() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				return setting.Value
			}
		}
	}
	return "unknown" // Or handle the case where info isn't available
}

// ProcessFlags kickstarts `dnsmonster`. it adds the basic module's flags
// checks their validity, sets up logging, metrics and loads input files
// associated with skipDomain and allowDomain
func ProcessFlags(ctx context.Context) {
	// todo: flags are camel-case but ini is not. this needs to be consistent

	iniParser := flags.NewIniParser(GlobalParser)
	GlobalParser.AddGroup("general", "General Options", &GeneralFlags)
	GlobalParser.AddGroup("help", "Help Options", &helpOptions)
	GlobalParser.AddGroup("metric", "Metrics", &globalMetricConfig)
	f, err := GlobalParser.Parse()
	if err != nil {
		log.Fatalf("Error parsing flags %v with error %s", f, err)
	}

	// process help options first
	if helpOptions.Help {
		GlobalParser.WriteHelp(os.Stdout)
		os.Exit(0)
	}
	if helpOptions.ManPage {
		GlobalParser.WriteManPage(os.Stdout)
		os.Exit(0)
	}
	if helpOptions.BashCompletion {
		fmt.Print(bashCompletionTemplate)
		os.Exit(0)
	}
	if helpOptions.SystemdService {
		fmt.Print(systemdServiceTemplate)
		os.Exit(0)
	}
	if helpOptions.FishCompletion {
		for _, g := range GlobalParser.Groups() {
			for _, arg := range g.Options() {
				fmt.Printf("complete -f -c dnsmonster -o -%s -d %#v\n", arg.LongName, arg.Description)
			}
		}
		os.Exit(0)
	}
	if helpOptions.WriteConfig != "" {
		iniParser.WriteFile(string(helpOptions.WriteConfig), flags.IniIncludeDefaults|flags.IniIncludeComments)
		os.Exit(0)
	}

	err = globalMetricConfig.SetupMetrics(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// check for config file option and parse it
	if GeneralFlags.Config != "" {
		err := iniParser.ParseFile(string(GeneralFlags.Config))
		if err != nil {
			if err != nil {
				log.Fatal(err)
			}
		}
		//  re-parse the argument from command line to give them priority
		_, err = GlobalParser.Parse()
		if err != nil {
			log.Fatal(err)
		}
	}

	var lvl log.Level = log.WarnLevel
	switch GeneralFlags.LogLevel {
	case 0:
		lvl = log.PanicLevel
	case 1:
		lvl = log.ErrorLevel
	case 2:
		lvl = log.WarnLevel
	case 3:
		lvl = log.InfoLevel
	case 4: // debug caller shows the function name
		lvl = log.DebugLevel
		log.SetReportCaller(true)
	}
	log.SetLevel(lvl)

	if GeneralFlags.Version {
		fmt.Printf("dnsmonster build %s, built at %s\n", GetCommitHash(), GetCommitDate())
		os.Exit(0)
	}

	switch GeneralFlags.LogFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text":
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}

	if GeneralFlags.SkipDomainsFile != "" {
		log.Info("skipDomainsFile is provided")
		GeneralFlags.LoadSkipDomain()
	}

	if GeneralFlags.AllowDomainsFile != "" {
		log.Info("allowDomainsFile is provided")
		GeneralFlags.LoadAllowDomain()
	}

	// ! show deprecation warning for skipDomainsFileType and allowDomainsFileType
	if GeneralFlags.SkipDomainsFileType != "" {
		log.Warn("skipDomainsFileType is a deprecated option and will be removed in future releases.")
	}
	if GeneralFlags.AllowDomainsFileType != "" {
		log.Warn("allowDomainsFileType is a deprecated option and will be removed in future releases.")
	}

	if GeneralFlags.MaskSize4 > 32 || GeneralFlags.MaskSize4 < 0 {
		log.Fatal("--maskSize4 must be between 0 and 32")
	}
	if GeneralFlags.MaskSize6 > 128 || GeneralFlags.MaskSize4 < 0 {
		log.Fatal("--maskSize6 must be between 0 and 128")
	}

	if GeneralFlags.PacketLimit < 0 {
		log.Fatal("--packetLimit must be equal or greather than 0")
	}
}

// vim: foldmethod=marker
