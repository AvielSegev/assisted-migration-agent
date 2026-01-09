import { useState } from "react";
import {
    Backdrop,
    Bullseye,
    Card,
    CardBody,
    CardHeader,
    Content,
    ContentVariants,
    Divider,
    Flex,
    FlexItem,
    Popover,
    Title,
} from "@patternfly/react-core";
import {
    InfoCircleIcon,
    OutlinedQuestionCircleIcon,
} from "@patternfly/react-icons";
import { useAppDispatch, useAppSelector } from "@shared/store";
import {
    Information,
    DataSharing,
    PrivacyNote,
    RedHatLogo,
} from "@shared/components";
import {
    startCollection,
    stopCollection,
} from "@shared/reducers/collectorSlice";
import { CollectorStatusStatusEnum } from "@generated/index";
import { Credentials } from "@models";
import VCenterLoginForm from "./VCenterLoginForm";
import CollectionProgress from "./VCenterLoginForm/CollectionProgress";

interface LoginProps {
    version?: string;
}

function Login({ version = "1.03" }: LoginProps) {
    const dispatch = useAppDispatch();
    const { status, loading, error } = useAppSelector(
        (state) => state.collector
    );

    const [isDataShared, setIsDataShared] = useState(false);
    const [collectionProgress, setCollectionProgress] = useState({
        percentage: 0,
        statusText: "Connecting...",
    });

    const isCollecting =
        status === CollectorStatusStatusEnum.Connecting ||
        status === CollectorStatusStatusEnum.Connected ||
        status === CollectorStatusStatusEnum.Collecting;

    const handleCollect = (credentials: Credentials) => {
        dispatch(
            startCollection({
                url: credentials.url,
                username: credentials.username,
                password: credentials.password,
            })
        );
    };

    const handleCancelCollection = () => {
        dispatch(stopCollection());
    };

    return (
        <>
            <Backdrop style={{ zIndex: 0 }} />
            <Bullseye style={{ minHeight: "100vh" }}>
                <Card
                    style={{
                        maxWidth: "36rem",
                        width: "100%",
                        maxHeight: "90vh",
                        overflowY: "auto",
                        borderRadius: "8px",
                    }}
                >
                    <CardHeader>
                        <Flex
                            direction={{ default: "column" }}
                            gap={{ default: "gapMd" }}
                        >
                            <FlexItem>
                                <RedHatLogo />
                            </FlexItem>

                            <Flex
                                justifyContent={{
                                    default: "justifyContentSpaceBetween",
                                }}
                            >
                                <FlexItem>
                                    <Title headingLevel="h1" size="2xl">
                                        Migration assessment
                                    </Title>
                                </FlexItem>
                                <FlexItem>
                                    <Content component={ContentVariants.small}>
                                        Agent ver. {version}
                                    </Content>
                                </FlexItem>
                            </Flex>

                            <FlexItem>
                                <Flex
                                    gap={{ default: "gapSm" }}
                                    alignItems={{ default: "alignItemsCenter" }}
                                >
                                    <FlexItem>
                                        <Content component={ContentVariants.p}>
                                            Migration Discovery VM
                                        </Content>
                                    </FlexItem>
                                    <FlexItem>
                                        <Popover bodyContent="The Migration Discovery VM collects infrastructure data from your vCenter environment to generate a migration assessment report.">
                                            <OutlinedQuestionCircleIcon
                                                style={{ color: "#000000" }}
                                            />
                                        </Popover>
                                    </FlexItem>
                                </Flex>
                            </FlexItem>

                            <FlexItem>
                                <Title headingLevel="h2" size="xl">
                                    vCenter login
                                </Title>
                            </FlexItem>

                            <FlexItem>
                                <Flex
                                    gap={{ default: "gapSm" }}
                                    alignItems={{
                                        default: "alignItemsFlexStart",
                                    }}
                                >
                                    <FlexItem>
                                        <InfoCircleIcon
                                            style={{ color: "#007bff" }}
                                        />
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Access control</strong>
                                    </FlexItem>
                                    <Flex
                                        direction={{ default: "column" }}
                                        gap={{ default: "gapXs" }}
                                    >
                                        <FlexItem>
                                            <Content
                                                component={ContentVariants.p}
                                            >
                                                A VMware user account with
                                                read-only permissions is
                                                sufficient for secure access
                                                during the discovery process.
                                            </Content>
                                        </FlexItem>
                                    </Flex>
                                </Flex>
                            </FlexItem>
                        </Flex>
                    </CardHeader>

                    <Divider
                        style={{
                            "--pf-v6-c-divider--Height": "8px",
                            "--pf-v6-c-divider--BackgroundColor": "#f5f5f5",
                        } as React.CSSProperties}
                    />

                    <CardBody>
                        <VCenterLoginForm
                            collect={handleCollect}
                            cancelCollection={handleCancelCollection}
                            isLoading={loading || isCollecting}
                            isDataShared={isDataShared}
                            dataSharingComponent={
                                <DataSharing
                                    variant="checkbox"
                                    isChecked={isDataShared}
                                    onChange={setIsDataShared}
                                    isDisabled={loading || isCollecting}
                                />
                            }
                            informationComponent={
                                <Information error={error}>
                                    <PrivacyNote />
                                </Information>
                            }
                            progressComponent={
                                <CollectionProgress
                                    percentage={collectionProgress.percentage}
                                    statusText={collectionProgress.statusText}
                                />
                            }
                        />
                    </CardBody>
                </Card>
            </Bullseye>
        </>
    );
}

export default Login;
