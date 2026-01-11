import { ReactNode } from "react";
import { Backdrop, Bullseye } from "@patternfly/react-core";

interface LayoutProps {
    children: ReactNode;
    variant?: "centered" | "full";
}

function Layout({ children, variant = "centered" }: LayoutProps) {
    if (variant === "full") {
        return (
            <div style={{ minHeight: "100vh", width: "100%" }}>
                {children}
            </div>
        );
    }

    // Centered layout (default) - for Login page
    return (
        <>
            <Backdrop style={{ zIndex: 0 }} />
            <Bullseye style={{ minHeight: "100vh" }}>{children}</Bullseye>
        </>
    );
}

export default Layout;
