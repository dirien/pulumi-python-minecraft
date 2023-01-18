import pulumi
from pulumi_aws import s3
from pulumi_aws import ec2, get_ami, GetAmiFilterArgs


minecraft_vpc = ec2.Vpc("minecraft-vpc",
                        cidr_block="10.0.0.0/16",
                        enable_dns_hostnames=True,
                        enable_dns_support=True,
                        )

minecraft_subnet = ec2.Subnet("minecraft-subnet",
                              vpc_id=minecraft_vpc.id,
                              cidr_block="10.0.48.0/20",
                              map_public_ip_on_launch=True,
                              availability_zone="eu-central-1a"
                              )

minecraft_internet_gateway = ec2.InternetGateway("minecraft-internet-gateway",
                                                 vpc_id=minecraft_vpc.id,
                                                 )

minecraft_route_table = ec2.RouteTable("minecraft-route-table",
                                       vpc_id=minecraft_vpc.id,
                                       routes=[{
                                           "cidr_block": "0.0.0.0/0",
                                           "gateway_id": minecraft_internet_gateway.id,
                                       }],
                                       )

minecraft_route_table_association = ec2.RouteTableAssociation("minecraft-route-table-association",
                                                              subnet_id=minecraft_subnet.id,
                                                              route_table_id=minecraft_route_table.id,
                                                              )

minecraft_security_group = ec2.SecurityGroup("minecraft-security-group",
                                             description="Allow Minecraft traffic",
                                             vpc_id=minecraft_vpc.id,
                                             ingress=[
                                                 ec2.SecurityGroupIngressArgs(
                                                     description="Minecraft",
                                                     protocol="tcp",
                                                     from_port=25565,
                                                     to_port=25565,
                                                     cidr_blocks=[
                                                         "0.0.0.0/0"
                                                     ]
                                                 ),
                                                 ec2.SecurityGroupIngressArgs(
                                                     description="SSH",
                                                     protocol="tcp",
                                                     from_port=22,
                                                     to_port=22,
                                                     cidr_blocks=[
                                                         "0.0.0.0/0"
                                                     ]
                                                 )
                                             ],
                                             egress=[
                                                 ec2.SecurityGroupEgressArgs(
                                                     description="All",
                                                     protocol="-1",
                                                     from_port=0,
                                                     to_port=0,
                                                     cidr_blocks=[
                                                         "0.0.0.0/0"
                                                     ]
                                                 )
                                             ],
                                             )

minecraft_ami = ec2.get_ami(filters=[
    GetAmiFilterArgs(
        name="name",
        values=[
            "ubuntu-minimal/images/hvm-ssd/ubuntu-jammy-22.04*"
        ],
    ),
    GetAmiFilterArgs(
        name="architecture",
        values=[
            "x86_64"
        ],
    )
], most_recent=True, owners=["099720109477"])


minecraft_keypair = ec2.KeyPair("minecraft-keypair",
                                key_name="minecraft",
                                public_key=open("minecraft.pub").read(),
                                )

minecraft_vm = ec2.Instance("minecraft-vm",
                            instance_type="t3.xlarge",
                            ami=minecraft_ami.id,
                            subnet_id=minecraft_subnet.id,
                            vpc_security_group_ids=[
                                minecraft_security_group.id
                            ],
                            key_name=minecraft_keypair.key_name,
                            user_data=open("cloud-init.yaml").read(),
                            )

pulumi.export("minecraft_vm_ip", minecraft_vm.public_ip)
