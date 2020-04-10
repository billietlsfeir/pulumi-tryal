package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v2/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Network
		network, err := compute.NewNetwork(ctx, "my-network", &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.BoolPtr(false),
		})
		if err != nil {
			return err
		}
		subnetwork, err := compute.NewSubnetwork(ctx, "my-network", &compute.SubnetworkArgs{
			Network:     network.Name,
			IpCidrRange: pulumi.String("10.0.0.0/24"),
		})
		if err != nil {
			return err
		}
		ctx.Export("subnetworkID", subnetwork.ID())
		publicAddress, err := compute.NewGlobalAddress(ctx, "my-network", &compute.GlobalAddressArgs{
			AddressType: pulumi.String("EXTERNAL"),
		})
		if err != nil {
			return err
		}
		ctx.Export("IPAddr", publicAddress.Address)
		// NetworkServices
		router, err := compute.NewRouter(ctx, "my-network", &compute.RouterArgs{
			Network: network.Name,
		})
		if err != nil {
			return err
		}
		ctx.Export("router", router.ID())
		routerNAT, err := compute.NewRouterNat(ctx, "my-network", &compute.RouterNatArgs{
			Router:                        router.Name,
			NatIpAllocateOption:           pulumi.String("AUTO_ONLY"),
			SourceSubnetworkIpRangesToNat: pulumi.String("ALL_SUBNETWORKS_ALL_IP_RANGES"),
		})
		if err != nil {
			return err
		}
		ctx.Export("routerNAT", routerNAT.ID())
		// Firewall
		firewallWeb, err := compute.NewFirewall(ctx, "my-network", &compute.FirewallArgs{
			Allows: compute.FirewallAllowArray{
				compute.FirewallAllowArgs{
					Ports: pulumi.StringArray{
						pulumi.String("22"),
						pulumi.String("80"),
					},
					Protocol: pulumi.String("tcp"),
				},
			},
			Direction: pulumi.String("INGRESS"),
			Network:   network.Name,
			TargetTags: pulumi.StringArray{
				pulumi.String("web"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("firewallWeb", firewallWeb.ID())
		firewallHealthcheck, err := compute.NewFirewall(ctx, "my-networkhealthcheck", &compute.FirewallArgs{
			Allows: compute.FirewallAllowArray{
				compute.FirewallAllowArgs{
					Ports: pulumi.StringArray{
						pulumi.String("80"),
					},
					Protocol: pulumi.String("tcp"),
				},
			},
			SourceRanges: pulumi.StringArray{
				pulumi.String("130.211.0.0/22"),
				pulumi.String("35.191.0.0/16"),
			},
			Direction: pulumi.String("INGRESS"),
			Network:   network.Name,
			TargetTags: pulumi.StringArray{
				pulumi.String("web"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("firewallHealthcheck", firewallHealthcheck.ID())
		// Compute instances
		MetadataStartupscript := `#!/bin/bash
		mkdir -p /var/www
		cd /var/www
		echo "<html><head><title>Bienvenue</title></head><body><h2>Bienvenue sur $(hostname)</h2></body></html>" > index.html
		nohup python3 -m http.server 80
		`
		instanceTemplate, err := compute.NewInstanceTemplate(ctx, "my-network", &compute.InstanceTemplateArgs{
			Disks: compute.InstanceTemplateDiskArray{
				compute.InstanceTemplateDiskArgs{
					Boot:        pulumi.Bool(true),
					SourceImage: pulumi.String("ubuntu-os-cloud/ubuntu-1804-bionic-v20200317"),
				},
			},
			MachineType: pulumi.String("f1-micro"),
			NetworkInterfaces: compute.InstanceTemplateNetworkInterfaceArray{
				compute.InstanceTemplateNetworkInterfaceArgs{
					Network:    network.Name,
					Subnetwork: subnetwork.Name,
				},
			},
			Tags: pulumi.StringArray{
				pulumi.String("web"),
			},
			MetadataStartupScript: pulumi.String(MetadataStartupscript),
		})
		if err != nil {
			return err
		}
		ctx.Export("instanceTemplate", instanceTemplate.ID())
		instanceGroup, err := compute.NewInstanceGroupManager(ctx, "my-network", &compute.InstanceGroupManagerArgs{
			BaseInstanceName: pulumi.String("http-server"),
			TargetSize:       pulumi.Int(3),
			Versions: compute.InstanceGroupManagerVersionArray{
				compute.InstanceGroupManagerVersionArgs{
					InstanceTemplate: instanceTemplate.SelfLink,
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("instanceGroup", instanceGroup.ID())

		// LoadBalancer
		loadBalancer, err := NewLoadBalancer(
			ctx,
			"my-network",
			&LoadBalancerArgs{
				Port:            80,
				HealthcheckPath: "/index.html",
				InstanceGroup:   *instanceGroup,
				PublicAddress:   *publicAddress,
			},
		)
		if err != nil {
			return err
		}
		ctx.Export("loadbalancer", loadBalancer.ID())
		return nil
	})
}
