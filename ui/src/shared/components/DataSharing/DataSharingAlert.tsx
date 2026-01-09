import { Alert, AlertVariant, Content } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

interface DataSharingAlertProps {
  onShare?: () => void;
}

function DataSharingAlert({ onShare }: DataSharingAlertProps) {
  return (
    <Alert
      variant={AlertVariant.info}
      isInline
      title="This report is not being shared with Red Hat"
    >
      <Content component="p">
        Unlock SaaS-only features.
        <br />
        By sharing aggregated data with Red Hat, you gain access to exclusive
        cloud capabilities and enhanced insights.
      </Content>
      <Content component="p">
        {onShare && (
          <a href="#" onClick={(e) => { e.preventDefault(); onShare(); }}>
            Share aggregated data with Red Hat
          </a>
        )}{' '}
        <a
          href="https://www.redhat.com/en/about/privacy-policy"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn more <ExternalLinkAltIcon />
        </a>
      </Content>
    </Alert>
  );
}

export default DataSharingAlert;
