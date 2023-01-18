
import unittest
import pulumi


class MyMinecraftMock(pulumi.runtime.Mocks):
    def new_resource(self, args: pulumi.runtime.MockResourceArgs):
        outputs = args.inputs
        if args.typ == "aws:ec2/instance:Instance":
            outputs = {
                **args.inputs,
            }
        return [args.name + '_id', outputs]

    def call(self, args: pulumi.runtime.MockCallArgs):
        print(args)
        if args.token == "aws:ec2/getAmi:getAmi":
            return {
                "architecture": "x86_64",
                "id": "ami-07866401f69c8006d",
            }
        return {}


pulumi.runtime.set_mocks(mocks=MyMinecraftMock())
import minecraft

class TestMinecraft(unittest.TestCase):

    @pulumi.runtime.test
    def test_minecraft(self):
        def check_instance_type(args):
            instance_type = args[0]
            self.assertEqual(instance_type, "t3.xlarge")

        return pulumi.Output.all(minecraft.minecraft_vm.instance_type).apply(check_instance_type)
