export type VMStatus = "migratable" | "migratable-with-warnings" | "not-migratable";

export interface VMIssue {
  type: "warning" | "error";
  message: string;
}

export interface VirtualMachine {
  id: string;
  name: string;
  status: VMStatus;
  datacenter: string;
  cluster: string;
  diskSizeGB: number;
  memorySizeGB: number;
  cpuCount: number;
  os?: string;
  powerState: string;
  issues: VMIssue[];
}

// Mock data for development - will be replaced with API data
export const mockVMs: VirtualMachine[] = [
  {
    id: "vm-1001",
    name: "web-server-01",
    status: "migratable",
    datacenter: "DC-East",
    cluster: "Production-Cluster-1",
    diskSizeGB: 240,
    memorySizeGB: 64,
    cpuCount: 8,
    os: "Red Hat Enterprise Linux 8",
    powerState: "poweredOn",
    issues: [],
  },
  {
    id: "vm-1002",
    name: "db-server-01",
    status: "migratable-with-warnings",
    datacenter: "DC-East",
    cluster: "Production-Cluster-1",
    diskSizeGB: 500,
    memorySizeGB: 128,
    cpuCount: 16,
    os: "Red Hat Enterprise Linux 7",
    powerState: "poweredOn",
    issues: [
      { type: "warning", message: "VM OS must be upgraded to be supported" },
    ],
  },
  {
    id: "vm-1003",
    name: "app-server-01",
    status: "not-migratable",
    datacenter: "DC-West",
    cluster: "Development-Cluster",
    diskSizeGB: 120,
    memorySizeGB: 32,
    cpuCount: 4,
    os: "Windows Server 2012",
    powerState: "poweredOn",
    issues: [
      { type: "error", message: "VM OS not migrate-able" },
    ],
  },
  {
    id: "vm-1004",
    name: "cache-server-01",
    status: "migratable",
    datacenter: "DC-East",
    cluster: "Production-Cluster-2",
    diskSizeGB: 80,
    memorySizeGB: 16,
    cpuCount: 4,
    os: "Ubuntu 22.04",
    powerState: "poweredOn",
    issues: [],
  },
  {
    id: "vm-1005",
    name: "monitoring-01",
    status: "migratable-with-warnings",
    datacenter: "DC-West",
    cluster: "Infrastructure-Cluster",
    diskSizeGB: 200,
    memorySizeGB: 32,
    cpuCount: 8,
    os: "CentOS 7",
    powerState: "poweredOff",
    issues: [
      { type: "warning", message: "VM is powered off" },
    ],
  },
];
