import React, { ReactNode } from "react";
import {
  Button,
  Content,
  Dropdown,
  DropdownItem,
  DropdownList,
  Flex,
  FlexItem,
  Icon,
  MenuToggle,
  MenuToggleElement,
  Split,
  SplitItem,
  Stack,
  StackItem,
  Title,
} from "@patternfly/react-core";
import {
  CheckCircleIcon,
  ExportIcon,
  ExternalLinkAltIcon,
} from "@patternfly/react-icons";

interface HeaderProps {
  totalVMs?: number;
  totalClusters?: number;
  isConnected?: boolean;
  lastUpdated?: string;
  onExport?: () => void;
  children?: ReactNode;
}

const Header: React.FC<HeaderProps> = ({
  totalVMs = 0,
  totalClusters = 0,
  isConnected = true,
  lastUpdated,
  onExport,
  children,
}) => {
  const [isExportOpen, setIsExportOpen] = React.useState(false);

  const onExportToggle = () => {
    setIsExportOpen(!isExportOpen);
  };

  return (
    <Stack hasGutter>
      <StackItem>
        <Split hasGutter>
          <SplitItem isFilled>
            <Flex direction={{ default: "column" }} gap={{ default: "gapSm" }}>
              <FlexItem>
                <Flex gap={{ default: "gapMd" }} alignItems={{ default: "alignItemsCenter" }}>
                  <FlexItem>
                    <Title headingLevel="h1" size="2xl">
                      Migration Assessment Report
                    </Title>
                  </FlexItem>
                  <FlexItem>
                    <a href="#" style={{ fontSize: "14px" }}>
                      Contact us
                    </a>
                  </FlexItem>
                </Flex>
              </FlexItem>

              <FlexItem>
                <Flex gap={{ default: "gapSm" }} alignItems={{ default: "alignItemsCenter" }}>
                  <FlexItem>
                    <Content component="small" style={{ fontWeight: 500 }}>
                      Discovery VM status:
                    </Content>
                  </FlexItem>
                  <FlexItem>
                    {isConnected ? (
                      <Flex gap={{ default: "gapXs" }} alignItems={{ default: "alignItemsCenter" }}>
                        <Icon status="success">
                          <CheckCircleIcon />
                        </Icon>
                        <Content component="small">Connected</Content>
                      </Flex>
                    ) : (
                      <Content component="small">Disconnected</Content>
                    )}
                  </FlexItem>
                </Flex>
              </FlexItem>

              <FlexItem>
                <Content component="p">
                  Presenting the information we were able to fetch from the discovery process
                </Content>
              </FlexItem>

              {lastUpdated && (
                <FlexItem>
                  <Content component="small" style={{ color: "#6a6e73" }}>
                    {lastUpdated}
                  </Content>
                </FlexItem>
              )}

              <FlexItem>
                <Content component="p">
                  Detected <strong>{totalVMs.toLocaleString()} VMs</strong> in{" "}
                  <strong>{totalClusters} clusters</strong>
                </Content>
              </FlexItem>
            </Flex>
          </SplitItem>

          <SplitItem>
            <Dropdown
              isOpen={isExportOpen}
              onSelect={() => setIsExportOpen(false)}
              onOpenChange={setIsExportOpen}
              toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                  ref={toggleRef}
                  onClick={onExportToggle}
                  isExpanded={isExportOpen}
                  variant="secondary"
                >
                  <ExportIcon /> Export report
                </MenuToggle>
              )}
            >
              <DropdownList>
                <DropdownItem key="pdf" onClick={onExport}>
                  Export as PDF
                </DropdownItem>
                <DropdownItem key="csv">
                  Export as CSV
                </DropdownItem>
              </DropdownList>
            </Dropdown>
          </SplitItem>
        </Split>
      </StackItem>

      {children && <StackItem>{children}</StackItem>}
    </Stack>
  );
};

Header.displayName = "Header";

export default Header;
