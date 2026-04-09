# Mirror-Cloud: Ephemeral AWS environment for testing infrastructure
# This mirrors the LocalStack setup but in a real AWS account
# Runs only when specifically activated via `cpctl env up mirror`
# Auto-teardown after TTL to avoid unexpected costs

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  # Use local state file for mirror environment
  # Can be swapped to S3 backend for team environments
  backend "local" {
    path = "terraform.tfstate.mirror"
  }
}

provider "aws" {
  profile = var.aws_profile
  region  = var.aws_region

  default_tags {
    tags = {
      Environment = "mirror"
      ManagedBy   = "cloud-playground"
      CreatedAt   = timestamp()
      TTL         = var.auto_teardown_ttl
    }
  }
}

# === VARIABLES ===

variable "aws_profile" {
  description = "AWS profile for mirror account (must be configured in ~/.aws/config)"
  type        = string
  default     = "mirror-account"
}

variable "aws_region" {
  description = "AWS region for mirror environment"
  type        = string
  default     = "eu-central-1"
}

variable "auto_teardown_ttl" {
  description = "Time-to-live before auto-teardown (e.g., '4h', '24h')"
  type        = string
  default     = "4h"
}

variable "vpc_cidr" {
  description = "VPC CIDR block for Mirror-Cloud"
  type        = string
  default     = "10.100.0.0/16"
}

variable "enable_compute" {
  description = "Enable EC2, ECS, Lambda"
  type        = bool
  default     = true
}

variable "enable_database" {
  description = "Enable RDS PostgreSQL"
  type        = bool
  default     = true
}

variable "enable_batch" {
  description = "Enable AWS Batch"
  type        = bool
  default     = true
}

# === VPC & NETWORKING ===

resource "aws_vpc" "mirror" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "mirror-playground-vpc"
  }
}

resource "aws_internet_gateway" "mirror" {
  vpc_id = aws_vpc.mirror.id

  tags = {
    Name = "mirror-igw"
  }
}

resource "aws_subnet" "public" {
  count                   = 2
  vpc_id                  = aws_vpc.mirror.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "mirror-public-subnet-${count.index + 1}"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.mirror.id

  route {
    cidr_block      = "0.0.0.0/0"
    gateway_id      = aws_internet_gateway.mirror.id
  }

  tags = {
    Name = "mirror-public-rt"
  }
}

resource "aws_route_table_association" "public" {
  count          = 2
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# === DATA SOURCE ===

data "aws_availability_zones" "available" {
  state = "available"
}

# === OUTPUTS ===

output "vpc_id" {
  description = "VPC ID for mirror environment"
  value       = aws_vpc.mirror.id
}

output "public_subnets" {
  description = "Public subnet IDs"
  value       = aws_subnet.public[*].id
}

output "auto_teardown_in" {
  description = "This environment will auto-teardown after this TTL"
  value       = var.auto_teardown_ttl
}
