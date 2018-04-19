# nomadgen
Generate nomad plan (hcl) from "simple" toml config

Make it easier for developers to create nomad plans which are valid for our infrastructure and can be used as a base to further modify.
This is rather specific for our environment (docker, ipv6 only, incoming dynamic firewall, nagios/contact metadata, datacenter constraints)

# example
For comments, see the [nomadgen.toml.example](https://github.com/42wim/nomadgen/blob/master/nomadgen.toml.example) file

## input
```toml
job="prefix-p-team-mattermost"
contact="team@example.com"
tier="production"

[[taskgroup]]
name="main"
count=4

[[task]]
taskgroup="main"
name="task1"
nagiossms="2"
nagiosmail="3"
image="docker.io/server:latest"
port=80
porttype="tcp"
cpu=1000
memory=2000
firewall="g/netscaler"

[[task]]
taskgroup="main"
name="task2"
image="docker.io/redis:latest"
args=["-json","-port 8080"]
volumes=["/net/blah:/abc"]
tags=["redis","tag"]
port=8080
porttype="http"
checkpath="/"
grace="90s"
cpu=500
memory=1000
firewall="g/netscaler"
```

## output
```hcl
"job" "prefix-p-team-mattermost" {
  "datacenters" = ["dc"]

  "meta" = {
    "contact" = "team@example.com"
  }

  "constraint" = {
    "attribute" = "${meta.role}"
    "value"     = "nomad"
  }

  "constraint" = {
    "attribute" = "${meta.tier}"
    "value"     = "production"
  }

  "update" = {
    "stagger"      = "10s"
    "max_parallel" = 1
  }

  "group" "main" {
    "count" = 4

    "constraint" = {
      "value"    = "true"
      "operator" = "distinct_hosts"
    }

    "constraint" = {
      "attribute" = "${meta.datacenter}"
      "value"     = "2"
      "operator"  = "distinct_property"
    }

    "restart" = {
      "interval" = "1m"
      "attempts" = 5
      "delay"    = "10s"
      "mode"     = "delay"
    }

    "task" "prefix-p-team-mattermost-task1" {
      "meta" = {
        "nagios_mail" = "3"
        "nagios_sms"  = "2"
      }

      "driver" = "docker"

      "config" = {
        "image"                  = "docker.io/server:latest"
        "advertise_ipv6_address" = true

        "logging" = {
          "type" = "journald"
        }
      }

      "service" = {
        "name"         = "prefix-p-team-mattermost-task1"
        "port"         = 80
        "address_mode" = "driver"

        "check" = {
          "name"          = "prefix-p-team-mattermost-task1-check"
          "port"          = 80
          "address_mode"  = "driver"
          "type"          = "tcp"
          "interval"      = "20s"
          "timeout"       = "10s"
          "check_restart" = {}
        }
      }

      "env" = {
        "FIREWALL_80" = "g/netscaler"
      }

      "resources" = {
        "memory" = 2000
        "cpu"    = 1000

        "network" = {
          "mbits" = 1
        }
      }
    }

    "task" "prefix-p-team-mattermost-task2" {
      "meta" = {
        "nagios_mail" = "-1"
        "nagios_sms"  = "-1"
      }

      "driver" = "docker"

      "config" = {
        "image"                  = "docker.io/redis:latest"
        "args"                   = ["-json", "-port 8080"]
        "advertise_ipv6_address" = true
        "volumes"                = ["/net/blah:/abc"]

        "logging" = {
          "type" = "journald"
        }
      }

      "service" = {
        "name"         = "prefix-p-team-mattermost-task2"
        "port"         = 8080
        "address_mode" = "driver"

        "check" = {
          "name"         = "prefix-p-team-mattermost-task2-check"
          "port"         = 8080
          "address_mode" = "driver"
          "type"         = "http"
          "path"         = "/"
          "interval"     = "20s"
          "timeout"      = "10s"

          "check_restart" = {
            "grace" = "90s"
          }
        }

        "tags" = ["redis", "tag"]
      }

      "env" = {
        "FIREWALL_8080" = "g/netscaler"
      }

      "resources" = {
        "memory" = 1000
        "cpu"    = 500

        "network" = {
          "mbits" = 1
        }
      }
    }
  }
}
```
