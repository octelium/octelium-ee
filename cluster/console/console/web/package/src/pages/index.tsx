import Footer from "@/components/Footer";

import TopBar from "@/components/TopBar";

import { Navigate, Outlet } from "react-router-dom";

import { useAppDispatch } from "@/utils/hooks";

import SideBar from "@/components/SideBar";
import { Toaster } from "@/components/ui/sonner";

import { AppShell, Burger } from "@mantine/core";
import { useDisclosure, useHeadroom } from "@mantine/hooks";

import RightSidebar from "@/components/SideBar/RightSidebar";
import { ScrollRestoration } from "react-router";

export default () => {
  const dispatch = useAppDispatch();

  const urlSearchParams = new URLSearchParams(window.location.search);
  if (urlSearchParams.get("redirect")) {
    const val = urlSearchParams.get("redirect")!;
    urlSearchParams.delete("redirect");
    return <Navigate to={val} />;
  }

  const [opened, { toggle }] = useDisclosure();
  const pinned = useHeadroom({ fixedAt: 120 });

  return (
    <div className="w-full !bg-slate-100">
      <title>Octelium Console</title>
      <ScrollRestoration />
      <div className=" bg-slate-100 min-h-screen antialiased">
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
          <AppShell.Header className="!bg-slate-100">
            <div className="flex flex-row items-center justify-center">
              <Burger
                opened={opened}
                onClick={toggle}
                hiddenFrom="sm"
                size="sm"
              />
              <TopBar />
            </div>
          </AppShell.Header>

          <AppShell.Navbar p="md" className="!bg-transparent mt-[60px]">
            <SideBar />
          </AppShell.Navbar>

          <AppShell.Main className="!bg-transparent h-full w-full mt-[60px]">
            <div className="flex-1 flex flex-col h-full w-full items-center justify-center">
              <div className="flex-1 w-full h-full">
                <Outlet />
              </div>
            </div>
          </AppShell.Main>
          <AppShell.Aside p="md" className="!bg-transparent">
            <RightSidebar />
          </AppShell.Aside>
        </AppShell>

        <Toaster
          position="bottom-center"
          toastOptions={{
            className: `bg-zinc-800 font-bold text-white`,
          }}
        />
      </div>
      <Footer />
    </div>
  );
};
