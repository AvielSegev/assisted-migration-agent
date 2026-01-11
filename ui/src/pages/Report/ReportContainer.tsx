import React, { useState } from "react";
import {
  MenuToggle,
  MenuToggleElement,
  Select,
  SelectList,
  SelectOption,
  Stack,
  StackItem,
  Tab,
  Tabs,
  TabTitleText,
} from "@patternfly/react-core";
import { useAppSelector } from "@shared/store";
import { Infra, InventoryData, VMs } from "@generated/index";
import { DataSharingAlert } from "@shared/components";
import Header from "./Header";
import Report from "./Report";
import { VMTable, mockVMs } from "./VirtualMachines";
import { buildClusterViewModel, ClusterOption } from "./assessment-report/clusterView";

const ReportContainer: React.FC = () => {
  const { inventory } = useAppSelector((state) => state.collector);
  const { mode } = useAppSelector((state) => state.agent);
  const [activeTab, setActiveTab] = useState<string | number>(0);
  const [selectedClusterId, setSelectedClusterId] = useState<string>("all");
  const [isClusterSelectOpen, setIsClusterSelectOpen] = useState(false);

  if (!inventory) {
    return null;
  }

  const infra = inventory?.vcenter?.infra as Infra | undefined;
  const vms = inventory?.vcenter?.vms as VMs | undefined;
  const clusters = inventory?.clusters as { [key: string]: InventoryData } | undefined;

  const clusterView = buildClusterViewModel({
    infra,
    vms,
    clusters,
    selectedClusterId,
  });

  const clusterSelectDisabled = clusterView.clusterOptions.length <= 1;

  const handleClusterSelect = (
    _event: React.MouseEvent<Element, MouseEvent> | undefined,
    value: string | number | undefined
  ): void => {
    if (typeof value === "string") {
      setSelectedClusterId(value);
    }
    setIsClusterSelectOpen(false);
  };

  const totalVMs = vms?.total ?? 0;
  const totalClusters = clusters ? Object.keys(clusters).length : 0;
  const isDataShared = mode === "connected";

  const handleShare = () => {
    // TODO: Implement share functionality
    console.log("Share data with Red Hat");
  };

  return (
    <div style={{ padding: "24px", width: "100%" }}>
      <Stack hasGutter>
        {/* Header */}
        <StackItem>
          <Header
            totalVMs={totalVMs}
            totalClusters={totalClusters}
            isConnected={true}
          />
        </StackItem>

        {/* Data Sharing Alert - shown when not shared */}
        {!isDataShared && (
          <StackItem>
            <DataSharingAlert onShare={handleShare} />
          </StackItem>
        )}

        {/* Cluster Selector */}
        <StackItem>
          <Select
            isScrollable
            isOpen={isClusterSelectOpen}
            selected={clusterView.selectionId}
            onSelect={handleClusterSelect}
            onOpenChange={(isOpen: boolean) => {
              if (!clusterSelectDisabled) setIsClusterSelectOpen(isOpen);
            }}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
              <MenuToggle
                ref={toggleRef}
                isExpanded={isClusterSelectOpen}
                onClick={() => {
                  if (!clusterSelectDisabled) {
                    setIsClusterSelectOpen((prev) => !prev);
                  }
                }}
                isDisabled={clusterSelectDisabled}
                style={{ minWidth: "250px" }}
              >
                {clusterView.selectionLabel}
              </MenuToggle>
            )}
          >
            <SelectList>
              {clusterView.clusterOptions.map((option: ClusterOption) => (
                <SelectOption key={option.id} value={option.id}>
                  {option.label}
                </SelectOption>
              ))}
            </SelectList>
          </Select>
        </StackItem>

        {/* Tabs */}
        <StackItem>
          <Tabs
            activeKey={activeTab}
            onSelect={(_event, tabIndex) => setActiveTab(tabIndex)}
          >
            <Tab eventKey={0} title={<TabTitleText>Overview</TabTitleText>}>
              <div
                style={{
                  marginTop: "24px",
                  width: "60%",
                  marginLeft: "auto",
                  marginRight: "auto",
                }}
              >
                <Report clusterView={clusterView} />
              </div>
            </Tab>
            <Tab eventKey={1} title={<TabTitleText>Virtual Machines</TabTitleText>}>
              <div style={{ marginTop: "24px" }}>
                <VMTable vms={mockVMs} />
              </div>
            </Tab>
          </Tabs>
        </StackItem>
      </Stack>
    </div>
  );
};

ReportContainer.displayName = "ReportContainer";

export default ReportContainer;
