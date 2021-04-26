# Generated by: tyk-ci/wf-gen
# Generated on: Mon 26 Apr 13:12:55 UTC 2021

# Generation commands:
# ./pr.zsh -title Update Cloudsmith url -branch cloudsmith-change-url -repos tyk,tyk-analytics,tyk-pump,tyk-sink,tyk-identity-broker,raava
# m4 -E -DxREPO=tyk-pump


data "terraform_remote_state" "integration" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = "base-prod"
    }
  }
}

output "tyk-pump" {
  value = data.terraform_remote_state.integration.outputs.tyk-pump
  description = "ECR creds for tyk-pump repo"
}

output "region" {
  value = data.terraform_remote_state.integration.outputs.region
  description = "Region in which the env is running"
}
