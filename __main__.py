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
public_addr = compute.GlobalAddress(
    "my-address",
    address_type="EXTERNAL"
)

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
healthcheck_firewall = compute.Firewall(
    "allow-healthcheck", 
    allows=[
        {
            "ports": [ "80" ], 
            "protocol": "tcp"
        }
    ], 
    source_ranges=[
        "35.191.0.0/16",
        "130.211.0.0/22"
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

instance_template = compute.InstanceTemplate(
    "my-instance-template",
    disks=[
        {
            "boot": True,
            "sourceImage": "ubuntu-os-cloud/ubuntu-1804-bionic-v20200317"
        }
    ],
    machine_type="f1-micro", 
    network_interfaces=[
        {
            "network": network.name,
            "subnetwork": subnet.name,
        }
    ], 
    tags=["web"],
    metadata_startup_script=metadata_startup_script
)
instance_group = compute.InstanceGroupManager(
    "my-instance-group",
    base_instance_name="http-server",
    target_size=3,
    versions=[
        {
            "instanceTemplate": instance_template.self_link
        }
    ]
)

# LoadBalancer
health_check = compute.HealthCheck(
    "my-health-check",
    http_health_check={
        "port": 80,
        "port_name": "http",
        "request_path": "/index.html"
    }
)
backend_service = compute.BackendService(
    "my-backend-service",
    backends=[
        {
            "group": instance_group.instance_group
        }
    ],
    health_checks=health_check.self_link,
    port_name="http"
)
url_map = compute.URLMap(
    "my-url-map",
    default_service=backend_service.self_link,
    host_rules=[
        {
            "path_matcher": "allpaths",
            "hosts": ["*"]
        }
    ],
    path_matchers=[
        {
            "name": "allpaths",
            "defaultService": backend_service.self_link,
        },
    ],
)
http_proxy = compute.TargetHttpProxy(
    "my-http-proxy",
    url_map=url_map.name
)

forwardingRule = compute.GlobalForwardingRule(
    "default-rule", 
    target=http_proxy.self_link,
    ip_address=public_addr.address,
    port_range="80"
)

pulumi.export("IPAddr", public_addr.address)
