# Pulumi Minecraft Server on AWS

This Pulumi code deploys a Java-based vanilla Minecraft server on AWS EC2.

Add this IP address `${outputs.minecraft_vm_ip}` to your Minecraft client to connect to the server

### Step by step instructions

1. Start your Minecraft client

2. Click on Multiplayer 
<img src="https://raw.githubusercontent.com/pulumi-demos/pulumi-python-minecraft/3ab89d76d1ce67aadcda07ffc78a465e48df6204/docs/add_0.png" width="400">

4. Click on Add Server 
<img src="https://raw.githubusercontent.com/pulumi-demos/pulumi-python-minecraft/3ab89d76d1ce67aadcda07ffc78a465e48df6204/docs/add_1.png" width="400">

5. Enter a name and the IP address `${outputs.minecraft_vm_ip}` of the server 
<img src="https://raw.githubusercontent.com/pulumi-demos/pulumi-python-minecraft/3ab89d76d1ce67aadcda07ffc78a465e48df6204/docs/add_2.png" width="400">

6. Click on Done and select the server 
<img src="https://raw.githubusercontent.com/pulumi-demos/pulumi-python-minecraft/3ab89d76d1ce67aadcda07ffc78a465e48df6204/docs/add_3.png" width="400">

#### Enjoy the game!
