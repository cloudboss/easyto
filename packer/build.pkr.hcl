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

variable "archive_bootloader" {
  type    = string
}

variable "archive_kernel" {
  type    = string
}

variable "archive_preinit" {
  type    = string
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

variable "exec_converter" {
  type    = string
}

variable "is_public" {
  type    = bool
  default = false
}

variable "root_vol" {
  type    = string
  default = "xvdf"
}

variable "root_vol_size" {
  type    = number
  default = 2
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

locals {
  remote_archive_bootloader   = "/tmp/${basename(var.archive_bootloader)}"
  remote_archive_kernel       = "/tmp/${basename(var.archive_kernel)}"
  remote_archive_preinit      = "/tmp/${basename(var.archive_preinit)}"
  remote_exec_converter       = "/tmp/${basename(var.exec_converter)}"
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
    device_name               = "/dev/xvda"
    source_device_name        = "/dev/${var.root_vol}"
    volume_size               = var.root_vol_size
    volume_type               = "gp2"
  }
  launch_block_device_mappings {
    delete_on_termination     = true
    device_name               = "/dev/${var.root_vol}"
    volume_size               = var.root_vol_size
    volume_type               = "gp2"
  }
}

build {
  sources                     = ["source.amazon-ebssurrogate.builder_ami"]

  provisioner "file" {
    destination               = local.remote_archive_bootloader
    source                    = var.archive_bootloader
  }
  provisioner "file" {
    destination               = local.remote_archive_kernel
    source                    = var.archive_kernel
  }
  provisioner "file" {
    destination               = local.remote_archive_preinit
    source                    = var.archive_preinit
  }
  provisioner "file" {
    destination               = local.remote_exec_converter
    source                    = var.exec_converter
  }
  provisioner "shell" {
    env                       = {
      ARCHIVE_BOOTLOADER      = local.remote_archive_bootloader
      ARCHIVE_KERNEL          = local.remote_archive_kernel
      ARCHIVE_PREINIT         = local.remote_archive_preinit
      EXEC_CONVERTER          = local.remote_exec_converter
      CONTAINER_IMAGE         = var.container_image
      ROOT_VOL                = "/dev/${var.root_vol}"
    }
    execute_command           = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    script                    = "provision"
  }
}
