import Footer from "@/components/Footer";
import SideBar from "@/components/SideBar";
import RightSidebar from "@/components/SideBar/RightSidebar";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { AppShell, Burger } from "@mantine/core";
import { useDisclosure, useHeadroom } from "@mantine/hooks";
import { ScrollRestoration } from "react-router";
import { Navigate, Outlet } from "react-router-dom";

export default () => {
  const urlSearchParams = new URLSearchParams(window.location.search);
  if (urlSearchParams.get("redirect")) {
    const val = urlSearchParams.get("redirect")!;
    urlSearchParams.delete("redirect");
    return <Navigate to={val} />;
  }

  const [opened, { toggle }] = useDisclosure();
  const pinned = useHeadroom({ fixedAt: 120 });

  return (
    <div className="w-full min-h-screen flex flex-col bg-slate-100">
      <title>Octelium Console</title>
      <ScrollRestoration />

      <div className="flex-1 flex flex-col bg-slate-100 antialiased">
        <AppShell
          className="!bg-transparent"
          header={{ height: 60, collapsed: !pinned, offset: false }}
          navbar={{
            width: 300,
            breakpoint: "sm",
            collapsed: { mobile: !opened },
          }}
          aside={{
            width: 300,
            breakpoint: "md",
            collapsed: { desktop: false, mobile: true },
          }}
          padding="md"
        >
          <AppShell.Header
            className="!bg-slate-100 border-b border-slate-200"
            style={{ zIndex: 200 }}
          >
            <div className="flex flex-row items-center h-full">
              <Burger
                opened={opened}
                onClick={toggle}
                hiddenFrom="sm"
                size="sm"
              />
              <TopBar />
            </div>
          </AppShell.Header>

          <AppShell.Navbar
            p="md"
            className="!bg-transparent"
            style={{ zIndex: 100, marginTop: 60 }}
          >
            <SideBar />
          </AppShell.Navbar>

          <AppShell.Main className="!bg-transparent" style={{ marginTop: 60 }}>
            <Outlet />
          </AppShell.Main>

          <AppShell.Aside
            p="md"
            className="!bg-transparent"
            style={{ zIndex: 100, marginTop: 60 }}
          >
            <RightSidebar />
          </AppShell.Aside>
        </AppShell>
      </div>

      <Footer />

      <Toaster
        position="bottom-center"
        toastOptions={{
          className: "bg-zinc-800 font-bold text-white",
        }}
      />
    </div>
  );
};
