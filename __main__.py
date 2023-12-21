import pulumi

import minecraft
import minecraft_component

"""
minecraftComponent = minecraft_component.MyMinecraftServer(
    "minecraft-component"
)

pulumi.export("public_ip", minecraftComponent.public_ip)
"""

pulumi.export("minecraft_vm_ip", minecraft.minecraft_vm.public_ip)
with open('./Pulumi.README.md') as f:
    pulumi.export('readme', f.read())