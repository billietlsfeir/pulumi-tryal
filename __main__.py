import pulumi
from pulumi_gcp import compute

# Network
network = compute.Network(
    "my-network", 
    auto_create_subnetworks=False, 
    name="my-network"
)
subnet = compute.Subnetwork(
    "my-subnet", 
    network=network, 
    ip_cidr_range="10.0.0.0/24", 
    name="my-subnet"
)
public_addr = compute.Address("my-address")

# Network services
router = compute.Router(
    "my-router", 
    network=network
)
nat = compute.RouterNat(
    "my-nat", 
    router=router.name, 
    nat_ip_allocate_option="AUTO_ONLY", 
    source_subnetwork_ip_ranges_to_nat="ALL_SUBNETWORKS_ALL_IP_RANGES"
)

# Firewall
firewall = compute.Firewall(
    "allow-http-in", 
    allows=[
        {
            "ports": [ "22", "80" ], 
            "protocol": "tcp"
        }
    ], 
    direction="INGRESS", 
    network=network.name, 
    target_tags=["web"]
)

# Compute instances
metadata_startup_script = """#!/bin/bash
mkdir -p /var/www
cd /var/www
echo "<html><head><title>Bienvenue</title></head><body><h2>Bienvenue sur $(hostname)</h2></body></html>" > index.html
nohup python3 -m http.server 80
"""
instance = compute.Instance(
    "httpserver0", 
    boot_disk={
        "initializeParams": {
            "image": "ubuntu-os-cloud/ubuntu-1804-bionic-v20200317"
        }
    }, 
    desired_status="RUNNING",
    machine_type="f1-micro", 
    network_interfaces=[
        {
            "network": network.name,
            "subnetwork": subnet.name,
            "accessConfigs": [{}]
        }
    ], 
    tags=["web"],
    metadata_startup_script=metadata_startup_script)

pulumi.export("IPAddr", public_addr.address)
