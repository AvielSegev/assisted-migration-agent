import { Alert, AlertVariant, List, ListItem } from '@patternfly/react-core';

function PrivacyNote() {
  return (
    <Alert
      variant={AlertVariant.info}
      isInline
      title="Note about Red Hat data privacy"
      component="h3"
    >
      <List isPlain>
        <ListItem>Red Hat does not store any non-aggregated data.</ListItem>
        <ListItem>Red Hat does not store your vCenter credentials in any way.</ListItem>
      </List>
    </Alert>
  );
}

export default PrivacyNote;
