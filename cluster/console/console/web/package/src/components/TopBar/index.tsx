/// <reference types="vite-plugin-svgr/client" />

import Logo from "@/assets/l03.svg?react";
import { setUseListSearch } from "@/features/settings/slice";
import { useAppDispatch, useAppSelector } from "@/utils/hooks";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import SearchList from "../SearchList";

const TopBar = () => {
  const navigate = useNavigate();
  const settings = useAppSelector((state) => state.settings);
  const picURL =
    settings.status?.session?.metadata?.picURL ??
    settings.status?.user?.metadata?.picURL;

  const location = useLocation();
  const dispatch = useAppDispatch();

  React.useEffect(() => {
    dispatch(
      setUseListSearch({
        useListSearch:
          location.pathname.startsWith(`/core/`) &&
          /^\/core\/[^\/]+$/.test(location.pathname),
      }),
    );
  }, [location.pathname, dispatch]);

  return (
    <>
      <nav className="w-full h-[60px] border-b-[0px] border-slate-300 flex px-4">
        <button
          className="flex-none flex items-center justify-center"
          onClick={() => {
            navigate("/");
          }}
        >
          <Link to={`/`}>
            <Logo className="w-[100px] md:w-[200px] h-auto" />
          </Link>
        </button>
        <div className="flex-grow">
          <div className="ml-[100px]">
            {settings.useListSearch && <SearchList />}
          </div>
        </div>
        <div className="flex-none flex items-center">
          <div className="flex items-center justify-center align-middle">
            <button
              className="w-10 h-10 rounded-full border-white border-2 text-gray-600 hover:text-gray-900 font-bold transition-all duration-300"
              onClick={() => {
                navigate(`/settings`);
              }}
            >
              {picURL ? (
                <img
                  className="rounded-full w-full h-full"
                  src={picURL}
                  alt="User pic"
                />
              ) : (
                <div className="rounded-full bg-sky-600 hover:bg-indigo-800 transition-all duration-300 w-full h-full"></div>
              )}
            </button>
          </div>
        </div>
      </nav>
    </>
  );
};

export default TopBar;
