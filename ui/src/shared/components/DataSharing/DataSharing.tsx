import DataSharingCheckbox from './DataSharingCheckbox';
import DataSharingAlert from './DataSharingAlert';

interface DataSharingCheckboxVariant {
  variant: 'checkbox';
  isChecked: boolean;
  onChange: (checked: boolean) => void;
  isDisabled?: boolean;
}

interface DataSharingAlertVariant {
  variant: 'alert';
  onShare?: () => void;
}

type DataSharingProps = DataSharingCheckboxVariant | DataSharingAlertVariant;

function DataSharing(props: DataSharingProps) {
  if (props.variant === 'checkbox') {
    return (
      <DataSharingCheckbox
        isChecked={props.isChecked}
        onChange={props.onChange}
        isDisabled={props.isDisabled}
      />
    );
  }

  return <DataSharingAlert onShare={props.onShare} />;
}

export default DataSharing;
