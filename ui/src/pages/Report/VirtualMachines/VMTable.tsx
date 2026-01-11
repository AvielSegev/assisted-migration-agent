import React, { useState, useMemo } from "react";
import {
  Button,
  Dropdown,
  DropdownItem,
  DropdownList,
  MenuToggle,
  MenuToggleCheckbox,
  MenuToggleElement,
  Pagination,
  SearchInput,
  Toolbar,
  ToolbarContent,
  ToolbarFilter,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
} from "@patternfly/react-core";
import {
  Table,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
  ThProps,
} from "@patternfly/react-table";
import {
  FilterIcon,
  EllipsisVIcon,
  ExclamationTriangleIcon,
  ExclamationCircleIcon,
} from "@patternfly/react-icons";
import { VirtualMachine, VMStatus } from "./types";

interface VMTableProps {
  vms: VirtualMachine[];
}

type SortableColumn = "name" | "status" | "datacenter" | "cluster" | "diskSizeGB" | "memorySizeGB";

const statusLabels: Record<VMStatus, string> = {
  "migratable": "Migratable",
  "migratable-with-warnings": "With warnings",
  "not-migratable": "Not migratable",
};

const VMTable: React.FC<VMTableProps> = ({ vms }) => {
  // Pagination state
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);

  // Search state
  const [searchValue, setSearchValue] = useState("");

  // Filter state
  const [statusFilter, setStatusFilter] = useState<VMStatus[]>([]);
  const [isStatusFilterOpen, setIsStatusFilterOpen] = useState(false);

  // Sort state
  const [activeSortIndex, setActiveSortIndex] = useState<number | null>(null);
  const [activeSortDirection, setActiveSortDirection] = useState<"asc" | "desc">("asc");

  // Selection state
  const [selectedVMs, setSelectedVMs] = useState<Set<string>>(new Set());

  // Row actions dropdown state
  const [openActionMenuId, setOpenActionMenuId] = useState<string | null>(null);

  // Column definitions
  const columns: { key: SortableColumn; label: string; sortable: boolean }[] = [
    { key: "name", label: "Name", sortable: true },
    { key: "status", label: "Status", sortable: true },
    { key: "datacenter", label: "Data center", sortable: true },
    { key: "cluster", label: "Cluster", sortable: true },
    { key: "diskSizeGB", label: "Disk size", sortable: true },
    { key: "memorySizeGB", label: "Memory size", sortable: true },
  ];

  // Filter and search VMs
  const filteredVMs = useMemo(() => {
    return vms.filter((vm) => {
      // Search filter
      if (searchValue && !vm.name.toLowerCase().includes(searchValue.toLowerCase())) {
        return false;
      }
      // Status filter
      if (statusFilter.length > 0 && !statusFilter.includes(vm.status)) {
        return false;
      }
      return true;
    });
  }, [vms, searchValue, statusFilter]);

  // Sort VMs
  const sortedVMs = useMemo(() => {
    if (activeSortIndex === null) return filteredVMs;

    const columnKey = columns[activeSortIndex].key;
    return [...filteredVMs].sort((a, b) => {
      const aValue = a[columnKey];
      const bValue = b[columnKey];

      if (typeof aValue === "string" && typeof bValue === "string") {
        return activeSortDirection === "asc"
          ? aValue.localeCompare(bValue)
          : bValue.localeCompare(aValue);
      }
      if (typeof aValue === "number" && typeof bValue === "number") {
        return activeSortDirection === "asc" ? aValue - bValue : bValue - aValue;
      }
      return 0;
    });
  }, [filteredVMs, activeSortIndex, activeSortDirection, columns]);

  // Paginate VMs
  const paginatedVMs = useMemo(() => {
    const start = (page - 1) * perPage;
    return sortedVMs.slice(start, start + perPage);
  }, [sortedVMs, page, perPage]);

  // Sort handler
  const getSortParams = (columnIndex: number): ThProps["sort"] => ({
    sortBy: {
      index: activeSortIndex ?? undefined,
      direction: activeSortDirection,
    },
    onSort: (_event, index, direction) => {
      setActiveSortIndex(index);
      setActiveSortDirection(direction);
    },
    columnIndex,
  });

  // Selection handlers
  const isAllSelected = paginatedVMs.length > 0 && paginatedVMs.every((vm) => selectedVMs.has(vm.id));
  const isSomeSelected = paginatedVMs.some((vm) => selectedVMs.has(vm.id));

  const onSelectAll = (isSelected: boolean) => {
    if (isSelected) {
      const newSelected = new Set(selectedVMs);
      paginatedVMs.forEach((vm) => newSelected.add(vm.id));
      setSelectedVMs(newSelected);
    } else {
      const newSelected = new Set(selectedVMs);
      paginatedVMs.forEach((vm) => newSelected.delete(vm.id));
      setSelectedVMs(newSelected);
    }
  };

  const onSelectVM = (vm: VirtualMachine, isSelected: boolean) => {
    const newSelected = new Set(selectedVMs);
    if (isSelected) {
      newSelected.add(vm.id);
    } else {
      newSelected.delete(vm.id);
    }
    setSelectedVMs(newSelected);
  };

  // Status filter handlers
  const onStatusFilterSelect = (status: VMStatus) => {
    if (statusFilter.includes(status)) {
      setStatusFilter(statusFilter.filter((s) => s !== status));
    } else {
      setStatusFilter([...statusFilter, status]);
    }
  };

  const clearStatusFilter = () => {
    setStatusFilter([]);
  };

  // Render status cell with icon
  const renderStatus = (vm: VirtualMachine) => {
    const hasWarnings = vm.issues.some((i) => i.type === "warning");
    const hasErrors = vm.issues.some((i) => i.type === "error");

    return (
      <span style={{ display: "flex", alignItems: "center", gap: "8px" }}>
        {hasErrors && (
          <ExclamationCircleIcon color="var(--pf-t--global--icon--color--status--danger--default)" />
        )}
        {hasWarnings && !hasErrors && (
          <ExclamationTriangleIcon color="var(--pf-t--global--icon--color--status--warning--default)" />
        )}
        {statusLabels[vm.status]}
      </span>
    );
  };

  // Render issues column
  const renderIssues = (vm: VirtualMachine) => {
    if (vm.issues.length === 0) return "—";
    return vm.issues.map((issue) => issue.message).join(", ");
  };

  // Format size
  const formatSize = (sizeGB: number) => `${sizeGB}GB`;

  return (
    <div>
      {/* Toolbar */}
      <Toolbar clearAllFilters={clearStatusFilter}>
        <ToolbarContent>
          <ToolbarGroup variant="filter-group">
            <ToolbarItem>
              <Dropdown
                isOpen={false}
                onSelect={() => {}}
                toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                  <MenuToggle
                    ref={toggleRef}
                    isExpanded={false}
                    splitButtonOptions={{
                      items: [
                        <MenuToggleCheckbox
                          id="select-all"
                          key="select-all"
                          aria-label="Select all"
                          isChecked={isAllSelected ? true : isSomeSelected ? null : false}
                          onChange={(checked) => onSelectAll(checked)}
                        />,
                      ],
                    }}
                  />
                )}
              >
                {/* Bulk selection options could go here */}
              </Dropdown>
            </ToolbarItem>
          </ToolbarGroup>

          <ToolbarGroup variant="filter-group">
            <ToolbarItem>
              <SearchInput
                placeholder="Find by name"
                value={searchValue}
                onChange={(_event, value) => setSearchValue(value)}
                onClear={() => setSearchValue("")}
                style={{ minWidth: "200px" }}
              />
            </ToolbarItem>

            <ToolbarToggleGroup toggleIcon={<FilterIcon />} breakpoint="xl">
              <ToolbarFilter
                chips={statusFilter.map((s) => statusLabels[s])}
                deleteChip={(_category, chip) => {
                  const status = Object.entries(statusLabels).find(
                    ([, label]) => label === chip
                  )?.[0] as VMStatus;
                  if (status) onStatusFilterSelect(status);
                }}
                deleteChipGroup={clearStatusFilter}
                categoryName="Status"
              >
                <Dropdown
                  isOpen={isStatusFilterOpen}
                  onSelect={() => {}}
                  onOpenChange={setIsStatusFilterOpen}
                  toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                    <MenuToggle
                      ref={toggleRef}
                      onClick={() => setIsStatusFilterOpen(!isStatusFilterOpen)}
                      isExpanded={isStatusFilterOpen}
                    >
                      <FilterIcon /> Filters
                    </MenuToggle>
                  )}
                >
                  <DropdownList>
                    {(Object.keys(statusLabels) as VMStatus[]).map((status) => (
                      <DropdownItem
                        key={status}
                        onClick={() => onStatusFilterSelect(status)}
                        isSelected={statusFilter.includes(status)}
                      >
                        {statusLabels[status]}
                      </DropdownItem>
                    ))}
                  </DropdownList>
                </Dropdown>
              </ToolbarFilter>
            </ToolbarToggleGroup>
          </ToolbarGroup>

          <ToolbarGroup>
            <ToolbarItem>
              <Button variant="secondary" isDisabled={selectedVMs.size === 0}>
                Send to deep inspection
              </Button>
            </ToolbarItem>
          </ToolbarGroup>

          <ToolbarItem variant="pagination" align={{ default: "alignEnd" }}>
            <Pagination
              itemCount={sortedVMs.length}
              perPage={perPage}
              page={page}
              onSetPage={(_event, newPage) => setPage(newPage)}
              onPerPageSelect={(_event, newPerPage) => {
                setPerPage(newPerPage);
                setPage(1);
              }}
              variant="top"
              isCompact
            />
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {/* Table */}
      <Table aria-label="Virtual machines table" variant="compact">
        <Thead>
          <Tr>
            <Th screenReaderText="Select" />
            {columns.map((column, index) => (
              <Th
                key={column.key}
                sort={column.sortable ? getSortParams(index) : undefined}
              >
                {column.label}
              </Th>
            ))}
            <Th>Issues</Th>
            <Th screenReaderText="Actions" />
          </Tr>
        </Thead>
        <Tbody>
          {paginatedVMs.map((vm) => (
            <Tr key={vm.id}>
              <Td
                select={{
                  rowIndex: 0,
                  onSelect: (_event, isSelected) => onSelectVM(vm, isSelected),
                  isSelected: selectedVMs.has(vm.id),
                }}
              />
              <Td dataLabel="Name">{vm.name}</Td>
              <Td dataLabel="Status">{renderStatus(vm)}</Td>
              <Td dataLabel="Data center">{vm.datacenter}</Td>
              <Td dataLabel="Cluster">{vm.cluster}</Td>
              <Td dataLabel="Disk size">{formatSize(vm.diskSizeGB)}</Td>
              <Td dataLabel="Memory size">{formatSize(vm.memorySizeGB)}</Td>
              <Td dataLabel="Issues">{renderIssues(vm)}</Td>
              <Td isActionCell>
                <Dropdown
                  isOpen={openActionMenuId === vm.id}
                  onSelect={() => setOpenActionMenuId(null)}
                  onOpenChange={(isOpen) => setOpenActionMenuId(isOpen ? vm.id : null)}
                  toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                    <MenuToggle
                      ref={toggleRef}
                      variant="plain"
                      onClick={() =>
                        setOpenActionMenuId(openActionMenuId === vm.id ? null : vm.id)
                      }
                      isExpanded={openActionMenuId === vm.id}
                    >
                      <EllipsisVIcon />
                    </MenuToggle>
                  )}
                  popperProps={{ position: "right" }}
                >
                  <DropdownList>
                    <DropdownItem key="inspect">Send to deep inspection</DropdownItem>
                    <DropdownItem key="details">View details</DropdownItem>
                  </DropdownList>
                </Dropdown>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </div>
  );
};

VMTable.displayName = "VMTable";

export default VMTable;
