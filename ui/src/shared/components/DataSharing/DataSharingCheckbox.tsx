import {
  Checkbox,
  Content,
  ContentVariants,
  Flex,
  FlexItem,
  Icon,
  Popover,
} from '@patternfly/react-core';
import { ExternalLinkAltIcon, OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

interface DataSharingCheckboxProps {
  isChecked: boolean;
  onChange: (checked: boolean) => void;
  isDisabled?: boolean;
}

function DataSharingCheckbox({ isChecked, onChange, isDisabled = false }: DataSharingCheckboxProps) {
  return (
    <Flex direction={{ default: 'column' }} gap={{ default: 'gapSm' }}>
      <FlexItem>
        <Flex gap={{ default: 'gapSm' }} alignItems={{ default: 'alignItemsCenter' }}>
          <FlexItem>
            <Checkbox
              id="data-sharing-checkbox"
              label="I agree to share aggregated data about my environment with Red Hat."
              isChecked={isChecked}
              onChange={(_event, checked) => onChange(checked)}
              isDisabled={isDisabled}
            />
          </FlexItem>
          <FlexItem>
            <Popover
              bodyContent="Aggregated data helps Red Hat improve migration tools and provide better recommendations for your environment."
            >
              <Icon isInline status="info">
                <OutlinedQuestionCircleIcon />
              </Icon>
            </Popover>
          </FlexItem>
        </Flex>
      </FlexItem>

      <FlexItem>
        <Content component={ContentVariants.small}>
          Aggregated data does not include names of virtual machines, hosts, clusters, data
          centers, datastores, disks and networks.{' '}
          <a
            href="https://www.redhat.com/en/about/privacy-policy"
            target="_blank"
            rel="noopener noreferrer"
          >
            Learn more <ExternalLinkAltIcon />
          </a>
        </Content>
      </FlexItem>
    </Flex>
  );
}

export default DataSharingCheckbox;
