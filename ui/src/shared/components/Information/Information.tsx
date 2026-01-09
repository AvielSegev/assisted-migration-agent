import { ReactNode } from 'react';
import { Alert, AlertVariant } from '@patternfly/react-core';

interface InformationProps {
  error: string | null;
  children: ReactNode;
}

function Information({ error, children }: InformationProps) {
  if (error) {
    return (
      <Alert variant={AlertVariant.danger} isInline title="Error">
        {error}
      </Alert>
    );
  }

  return <>{children}</>;
}

export default Information;
