package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v2/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
)

// LoadBalancer a
type LoadBalancer struct {
	pulumi.ComponentResource

	HealthCheck          *compute.HealthCheck
	BackendService       *compute.BackendService
	URLMap               *compute.URLMap
	TargetHTTPProxy      *compute.TargetHttpProxy
	GlobalForwardingRule *compute.GlobalForwardingRule
}

// LoadBalancerArgs a
type LoadBalancerArgs struct {
	Port            int
	HealthcheckPath string
	PublicAddress   compute.GlobalAddress
	InstanceGroup   compute.InstanceGroupManager
}

// ID a
func (lb *LoadBalancer) ID() pulumi.Input {
	return pulumi.String("foo")
}

// URN a
func (lb LoadBalancer) URN() pulumi.URNOutput {
	return pulumi.URNOutput{
		OutputState: &pulumi.OutputState{},
	}
}

// NewLoadBalancer a
func NewLoadBalancer(ctx *pulumi.Context, name string, args *LoadBalancerArgs, opts ...pulumi.ResourceOption) (*LoadBalancer, error) {
	var result LoadBalancer
	ctx.RegisterComponentResource("SFEIR:GCPHelpers:LoadBalancer", name, result, opts...)

	healthCheck, err := compute.NewHealthCheck(
		ctx,
		fmt.Sprintf("%s-healthcheck", name),
		&compute.HealthCheckArgs{
			HttpHealthCheck: compute.HealthCheckHttpHealthCheckArgs{
				Port:        pulumi.IntPtr(args.Port),
				RequestPath: pulumi.String(args.HealthcheckPath),
			},
		},
		// pulumi.Parent(result),
	)
	if err != nil {
		return &result, err
	}
	result.HealthCheck = healthCheck
	// ctx.RegisterComponentResource("SFEIR:GCPHelpers:LoadBalancer", name, healthCheck, pulumi.Parent(result))

	backendService, err := compute.NewBackendService(
		ctx,
		fmt.Sprintf("%s-backendservice", name),
		&compute.BackendServiceArgs{
			Backends: compute.BackendServiceBackendArray{
				compute.BackendServiceBackendArgs{
					Group: args.InstanceGroup.InstanceGroup,
				},
			},
			HealthChecks: healthCheck.SelfLink,
			PortName:     pulumi.String("http"),
		},
	)
	if err != nil {
		return &result, err
	}
	result.BackendService = backendService

	urlMap, err := compute.NewURLMap(
		ctx,
		fmt.Sprintf("%s-urlmap", name),
		&compute.URLMapArgs{
			DefaultService: backendService.SelfLink,
			HostRules: compute.URLMapHostRuleArray{
				compute.URLMapHostRuleArgs{
					PathMatcher: pulumi.String("allpaths"),
					Hosts: pulumi.StringArray{
						pulumi.String("*"),
					},
				},
			},
			PathMatchers: compute.URLMapPathMatcherArray{
				compute.URLMapPathMatcherArgs{
					Name:           pulumi.String("allpaths"),
					DefaultService: backendService.SelfLink,
				},
			},
		},
	)
	if err != nil {
		return &result, err
	}
	result.URLMap = urlMap

	targetHTTPProxy, err := compute.NewTargetHttpProxy(
		ctx,
		fmt.Sprintf("%s-targethttpproxy", name),
		&compute.TargetHttpProxyArgs{
			UrlMap: urlMap.Name,
		},
	)
	if err != nil {
		return &result, err
	}
	result.TargetHTTPProxy = targetHTTPProxy

	globalForwardingRule, err := compute.NewGlobalForwardingRule(
		ctx,
		fmt.Sprintf("%s-globalforwardingrule", name),
		&compute.GlobalForwardingRuleArgs{
			Target:    targetHTTPProxy.SelfLink,
			IpAddress: args.PublicAddress.Address,
			PortRange: pulumi.String(fmt.Sprintf("%d", args.Port)),
		},
	)
	if err != nil {
		return &result, err
	}
	result.GlobalForwardingRule = globalForwardingRule

	return &result, nil
}
