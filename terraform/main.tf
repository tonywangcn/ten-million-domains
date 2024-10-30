terraform {
  required_providers {
    spot = {
      source = "rackerlabs/spot"
    }
  }
}

variable "rackspace_spot_token" {
  description = "Rackspace Spot authentication token"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "Region for the Spot instance"
  type        = string
  default     = "us-east-iad-1"
}


provider "spot" {
  token = var.rackspace_spot_token
}


# Extract the last three alphabetic characters from the region
locals {
  region_suffix = split("-", var.region)[2]
}

resource "spot_cloudspace" "cluster" {
  cloudspace_name    = "ten-million-domains-${local.region_suffix}"
  region             = var.region
  hacontrol_plane    = false
  wait_until_ready   = true
  timeouts = {
    create = "30m"
  }
  deployment_type = "gen2"
}

resource "spot_spotnodepool" "gp-1" {
  cloudspace_name = spot_cloudspace.cluster.cloudspace_name
  server_class    = "gp.vs1.2xlarge-${local.region_suffix}"
  bid_price       = 0.005
  desired_server_count = 10
}

resource "spot_spotnodepool" "gp-2" {
  cloudspace_name = spot_cloudspace.cluster.cloudspace_name
  server_class    = "gp.vs1.xlarge-${local.region_suffix}"
  bid_price       = 0.005
  desired_server_count = 10
}

resource "spot_spotnodepool" "ch-1" {
  cloudspace_name = spot_cloudspace.cluster.cloudspace_name
  server_class    = "ch.vs1.2xlarge-${local.region_suffix}"
  bid_price       = 0.005
  desired_server_count = 10
}

resource "spot_spotnodepool" "ch-2" {
  cloudspace_name = spot_cloudspace.cluster.cloudspace_name
  server_class    = "ch.vs1.xlarge-${local.region_suffix}"
  bid_price       = 0.003
  desired_server_count = 10
}


data "spot_kubeconfig" "cluster" {
  cloudspace_name = resource.spot_cloudspace.cluster.name
}

# Output kubeconfig to local file
resource "local_file" "kubeconfig" {
  filename = "${path.module}/kubeconfig-${spot_cloudspace.cluster.cloudspace_name}.yaml"  # Path to the file with the cluster name
  content  = data.spot_kubeconfig.cluster.raw  # The kubeconfig content
}