"""OPC-UA simulator for development and CI testing.

Provides a realistic OPC-UA server with industrial-like nodes:
- Temperature: sine wave (20-80°C)
- Pressure: random walk (1-10 bar)
- MachineStatus: cycles through Off(0), Running(1), Error(2)
- ProductionCount: monotonically increasing counter
"""

import asyncio
import math
import random
import time

from asyncua import Server, ua


async def main():
    server = Server()
    await server.init()
    server.set_endpoint("opc.tcp://0.0.0.0:4840")
    server.set_server_name("OPC-UA Simulator")

    uri = "urn:opcua-kube-gateway:simulator"
    idx = await server.register_namespace(uri)

    objects = server.nodes.objects
    device = await objects.add_object(idx, "SimulatedPLC")

    temperature = await device.add_variable(
        ua.NodeId("Temperature", idx), "Temperature", 20.0
    )
    pressure = await device.add_variable(
        ua.NodeId("Pressure", idx), "Pressure", 5.0
    )
    machine_status = await device.add_variable(
        ua.NodeId("MachineStatus", idx), "MachineStatus", 0
    )
    production_count = await device.add_variable(
        ua.NodeId("ProductionCount", idx), "ProductionCount", 0
    )

    await temperature.set_writable()
    await pressure.set_writable()
    await machine_status.set_writable()
    await production_count.set_writable()

    print(f"OPC-UA Simulator running at opc.tcp://0.0.0.0:4840")
    print(f"Namespace index: {idx}")
    print(f"Nodes:")
    print(f"  ns={idx};s=Temperature   (Double, sine wave 20-80°C)")
    print(f"  ns={idx};s=Pressure      (Double, random walk 1-10 bar)")
    print(f"  ns={idx};s=MachineStatus (Int32, cycles 0/1/2)")
    print(f"  ns={idx};s=ProductionCount (Int64, counter)")

    counter = 0
    pressure_val = 5.0
    status_cycle = [0, 1, 1, 1, 1, 1, 1, 1, 2, 1]
    status_idx = 0

    async with server:
        while True:
            elapsed = time.time()

            # Temperature: sine wave between 20 and 80°C, period ~60s
            temp_val = 50.0 + 30.0 * math.sin(elapsed * 2 * math.pi / 60.0)
            await temperature.write_value(round(temp_val, 2))

            # Pressure: bounded random walk between 1 and 10 bar
            pressure_val += random.uniform(-0.3, 0.3)
            pressure_val = max(1.0, min(10.0, pressure_val))
            await pressure.write_value(round(pressure_val, 2))

            # MachineStatus: cycle through states
            await machine_status.write_value(status_cycle[status_idx % len(status_cycle)])
            status_idx += 1

            # ProductionCount: monotonically increasing
            counter += random.randint(1, 5)
            await production_count.write_value(counter)

            await asyncio.sleep(1)


if __name__ == "__main__":
    asyncio.run(main())
