from pulumi_policy import (
    EnforcementLevel,
    PolicyPack,
    ReportViolation,
    ResourceValidationArgs,
    ResourceValidationPolicy,
)

def ec2_instance_type_validator(args: ResourceValidationArgs, report_violation: ReportViolation):
    if args.resource_type == "aws:ec2/instance:Instance" and "instanceType" in args.props:
        instance_type = args.props["instanceType"]
        if instance_type == "t2.micro":
            report_violation(
                "t2.micro instance types are too small to run Minecraft. " +
                "Please use a different instance type.")


ec2_instance_type = ResourceValidationPolicy(
    name="ec2-instance-type",
    description="Prohibits t2.micro EC2 instance types.",
    validate=ec2_instance_type_validator,
)

PolicyPack(
    name="aws-python",
    enforcement_level=EnforcementLevel.MANDATORY,
    policies=[
        ec2_instance_type,
    ],
)
