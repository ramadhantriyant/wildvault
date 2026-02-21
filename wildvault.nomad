job "wildvault" {
  datacenters = ["dc1"]
  type        = "batch"

  # Run at midnight on the 1st of every 2 months (Jan, Mar, May, Jul, Sep, Nov)
  periodic {
    crons            = ["0 0 1 */2 *"]
    prohibit_overlap = true
    time_zone        = "UTC"
  }

  group "cert-renewal" {
    count = 1

    restart {
      attempts = 3
      delay    = "30s"
      interval = "10m"
      mode     = "fail"
    }

    # Nomad injects VAULT_TOKEN automatically using this policy
    vault {
      policies = ["wildvault"]
    }

    task "wildvault" {
      driver = "exec"

      config {
        command = "local/wildvault"
      }

      # Download the pre-built binary before running
      artifact {
        source      = "https://git.ramadhantriyant.id/ramadhantriyant/wildvault/releases/download/latest/wildvault"
        destination = "local/wildvault"
        mode        = "file"
      }

      # VAULT_ADDR is required by vault-client-go (via vault.WithEnvironment())
      # VAULT_TOKEN is injected automatically by Nomad via the vault stanza above
      template {
        data        = <<-EOH
          VAULT_ADDR={{ with nomadVar "nomad/jobs/wildvault" }}{{ .vault_addr }}{{ end }}
        EOH
        destination = "secrets/env"
        env         = true
      }

      resources {
        cpu    = 100
        memory = 128
      }
    }
  }
}
