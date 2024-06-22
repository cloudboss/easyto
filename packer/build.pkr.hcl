packer {
  required_plugins {
    amazon    = {
      version = "= 1.2.6"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "ami_name" {
  type    = string
}

variable "architecture" {
  type    = string
  default = "x86_64"
}

variable "asset_dir" {
  type    = string
}

variable "asset_files" {
  type    = list(string)

  default = [
    "boot.tar",
    "chrony.tar",
    "ctr2disk",
    "init.tar",
    "kernel.tar",
    "ssh.tar",
  ]
}

variable "builder_ami_owner" {
  type    = string
  default = "136693071363"
}

variable "builder_ami_pattern" {
  type    = string
  default = "debian-12-*"
}

variable "builder_instance_type" {
  type    = string
  default = "t3.micro"
}

variable "container_image" {
  type    = string
}

variable "is_public" {
  type    = bool
  default = false
}

variable "login_user" {
  type    = string
  default = "cloudboss"
}

variable "login_shell" {
  type    = string
}

variable "root_device_name" {
  type    = string
}

variable "root_vol_size" {
  type    = number
  default = 2
}

variable "services" {
  type    = list(string)
}

variable "ssh_interface" {
  type    = string
  default = "public_ip"
}

variable "ssh_username" {
  type    = string
  default = "admin"
}

variable "subnet_id" {
  type    = string
}

variable "debug" {
  type    = bool
}

locals {
  remote_asset_dir        = "/tmp/assets"
  source_root_device_name = "/dev/xvdf"
}

data "amazon-ami" "builder_ami" {
  filters                     = {
    architecture              = "x86_64"
    name                      = var.builder_ami_pattern
    root-device-type          = "ebs"
    virtualization-type       = "hvm"
  }
  most_recent                 = true
  owners                      = [var.builder_ami_owner]
}

source "amazon-ebssurrogate" "builder_ami" {
  ami_description             = var.ami_name
  ami_groups                  = var.is_public ? ["all"] : []
  ami_name                    = var.ami_name
  ami_architecture            = var.architecture
  ami_virtualization_type     = "hvm"
  boot_mode                   = "uefi"
  associate_public_ip_address = var.ssh_interface == "public_ip"
  ena_support                 = true
  instance_type               = var.builder_instance_type
  run_tags                    = {
    Name                      = "ami-builder-${var.ami_name}"
  }
  run_volume_tags             = {
    Name                      = "ami-volume-${var.ami_name}"
  }
  snapshot_groups             = var.is_public ? ["all"] : []
  source_ami                  = data.amazon-ami.builder_ami.id
  sriov_support               = true
  ssh_interface               = var.ssh_interface
  ssh_pty                     = true
  ssh_timeout                 = "5m"
  ssh_username                = var.ssh_username
  subnet_id                   = var.subnet_id
  tags                        = {
    "container_image"         = var.container_image
  }

  ami_root_device {
    delete_on_termination     = true
    device_name               = var.root_device_name
    source_device_name        = local.source_root_device_name
    volume_size               = var.root_vol_size
    volume_type               = "gp2"
  }
  launch_block_device_mappings {
    delete_on_termination     = true
    device_name               = local.source_root_device_name
    volume_size               = var.root_vol_size
    volume_type               = "gp2"
  }
}

build {
  sources                     = ["source.amazon-ebssurrogate.builder_ami"]

  provisioner "file" {
    destination               = "/tmp"
    source                    = var.asset_dir
  }
  provisioner "shell" {
    env                       = {
      ASSET_DIR               = "/tmp/assets"
      ASSET_FILES             = join(" ", var.asset_files)
      CONTAINER_IMAGE         = var.container_image
      EXEC_CTR2DISK           = "/tmp/assets/ctr2disk"
      ROOT_DEVICE             = local.source_root_device_name
      SERVICES                = join(",", var.services)
      LOGIN_USER              = var.login_user
      LOGIN_SHELL             = var.login_shell
      DEBUG                   = var.debug
    }
    execute_command           = "sudo env {{ .Vars }} {{ .Path }}"
    script                    = "provision"
  }
}
