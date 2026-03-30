import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { SSHSession } from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import React, { useEffect, useRef, useState } from "react";

import { ActionIcon } from "@mantine/core";
import { FaPlay, FaStop } from "react-icons/fa";
import { RiResetLeftFill } from "react-icons/ri";

interface SSHFrame {
  content: Uint8Array<ArrayBufferLike>;
  timestamp: number;
}

interface APIResponse {
  frames: SSHFrame[];
  hasMore: boolean;
  nextPage?: number;
}

interface XTermSSHReplayProps {
  sshSession: SSHSession;

  initialPage?: number;
  useMockData?: boolean;
}

const generateMockPage = (page: number): APIResponse => {
  const encoder = new TextEncoder();
  const baseTime = 1000000000000;

  const mockSessions = [
    [
      { text: "\x1b[32muser@server:~$\x1b[0m ", delay: 0 },
      { text: "l", delay: 150 },
      { text: "s", delay: 100 },
      { text: " ", delay: 80 },
      { text: "-", delay: 120 },
      { text: "l", delay: 90 },
      { text: "a\r\n", delay: 200 },
      { text: "total 48\r\n", delay: 50 },
      {
        text: "drwxr-xr-x  5 user user 4096 Dec 17 10:30 \x1b[34m.\x1b[0m\r\n",
        delay: 20,
      },
      {
        text: "drwxr-xr-x  3 root root 4096 Dec 10 08:15 \x1b[34m..\x1b[0m\r\n",
        delay: 20,
      },
    ],
    [
      {
        text: "-rw-r--r--  1 user user  220 Dec 10 08:15 .bash_logout\r\n",
        delay: 20,
      },
      {
        text: "-rw-r--r--  1 user user 3526 Dec 10 08:15 .bashrc\r\n",
        delay: 20,
      },
      {
        text: "drwxr-xr-x  3 user user 4096 Dec 15 14:20 \x1b[34mDocuments\x1b[0m\r\n",
        delay: 20,
      },
      {
        text: "drwxr-xr-x  2 user user 4096 Dec 16 09:45 \x1b[34mDownloads\x1b[0m\r\n",
        delay: 20,
      },
      { text: "\x1b[32muser@server:~$\x1b[0m ", delay: 100 },
    ],
    [
      { text: "c", delay: 200 },
      { text: "d", delay: 120 },
      { text: " ", delay: 100 },
      { text: "D", delay: 150 },
      { text: "o", delay: 90 },
      { text: "c", delay: 110 },
      { text: "u", delay: 95 },
      { text: "m", delay: 100 },
      { text: "e", delay: 85 },
      { text: "n", delay: 90 },
      { text: "t", delay: 95 },
      { text: "s", delay: 100 },
      { text: "\r\n", delay: 200 },
      { text: "\x1b[32muser@server:~/Documents$\x1b[0m ", delay: 50 },
      { text: "c", delay: 180 },
      { text: "a", delay: 110 },
      { text: "t", delay: 100 },
      { text: " ", delay: 120 },
      { text: "r", delay: 90 },
      { text: "e", delay: 95 },
      { text: "a", delay: 100 },
      { text: "d", delay: 85 },
      { text: "m", delay: 90 },
      { text: "e", delay: 95 },
      { text: ".", delay: 100 },
      { text: "t", delay: 80 },
      { text: "x", delay: 85 },
      { text: "t", delay: 90 },
      { text: "\r\n", delay: 150 },
      { text: "=================================\r\n", delay: 30 },
      { text: "Welcome to the SSH Replay Demo!\r\n", delay: 30 },
      { text: "=================================\r\n", delay: 30 },
      { text: "\r\n", delay: 50 },
      { text: "This is a demonstration of real-time\r\n", delay: 30 },
      { text: "SSH session replay with xterm.js.\r\n", delay: 30 },
      { text: "\x1b[32muser@server:~/Documents$\x1b[0m ", delay: 200 },
      { text: "exit\r\n", delay: 500 },
      { text: "logout\r\n", delay: 100 },
      { text: "\x1b[33mConnection to server closed.\x1b[0m\r\n", delay: 50 },
    ],
  ];

  if (page < 1 || page > mockSessions.length) {
    return { frames: [], hasMore: false };
  }

  const session = mockSessions[page - 1];
  let currentTime = baseTime + (page - 1) * 10000;

  const frames: SSHFrame[] = session.map((item) => {
    currentTime += item.delay;
    return {
      content: encoder.encode(item.text),
      timestamp: currentTime,
    };
  });

  return {
    frames,
    hasMore: page < mockSessions.length,
    nextPage: page < mockSessions.length ? page + 1 : undefined,
  };
};

export const XTermSSHReplay: React.FC<XTermSSHReplayProps> = ({
  initialPage = 1,
  sshSession,
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const fitAddon = useRef<FitAddon | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [status, setStatus] = useState("Ready to play");
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!terminalRef.current) return;

    terminal.current = new Terminal({
      convertEol: true,
      fontFamily: "Ubuntu Mono",
      fontWeight: 600,
      fontWeightBold: 700,
      cursorBlink: true,
      scrollback: 1000,
      fontSize: 18,
    });

    fitAddon.current = new FitAddon();
    terminal.current.loadAddon(fitAddon.current);
    terminal.current.open(terminalRef.current);

    fitAddon.current.fit();

    const handleResize = () => {
      fitAddon.current?.fit();
    };
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      terminal.current?.dispose();
    };
  }, []);

  const fetchPage = async (
    page: number,
    signal: AbortSignal,
  ): Promise<APIResponse> => {
    if (isDev()) {
      await new Promise((resolve) => setTimeout(resolve, 300));
      if (signal.aborted) throw new Error("Aborted");
      return generateMockPage(page);
    }

    {
      const { response } =
        await getClientVisibilityAccessLog().listSSHSessionRecording({
          sessionID: sshSession.id,
          page: page - 1,
        });
      return {
        frames: response.items.map((x) => ({
          content: x.data,
          timestamp: Timestamp.toDate(x.timestamp!).getTime(),
        })),
        hasMore: response.listResponseMeta?.hasMore ?? false,
        nextPage: page,
      };
    }
  };

  const sleep = (ms: number) =>
    new Promise((resolve) => setTimeout(resolve, ms));

  const playSession = async () => {
    if (!terminal.current || isPlaying) return;

    setIsPlaying(true);
    setStatus("Loading...");
    terminal.current.clear();

    abortController.current = new AbortController();
    const signal = abortController.current.signal;

    try {
      let currentPage = initialPage;
      let allFrames: SSHFrame[] = [];
      let hasMore = true;

      while (hasMore) {
        if (signal.aborted) break;

        setStatus(`Fetching page ${currentPage}...`);
        const response = await fetchPage(currentPage, signal);

        allFrames = [...allFrames, ...response.frames];
        hasMore = response.hasMore;

        if (hasMore && response.nextPage) {
          currentPage = response.nextPage;
        } else if (hasMore) {
          currentPage++;
        }
      }

      if (signal.aborted) {
        setStatus("Playback stopped");
        setIsPlaying(false);
        return;
      }

      allFrames.sort((a, b) => a.timestamp - b.timestamp);

      setStatus(`Playing ${allFrames.length} frames...`);

      let prevTimestamp = allFrames[0]?.timestamp || 0;

      for (let i = 0; i < allFrames.length; i++) {
        if (signal.aborted) break;

        const frame = allFrames[i];
        const delay = frame.timestamp - prevTimestamp;

        if (delay > 0 && i > 0) {
          await sleep(delay);
        }

        terminal.current.write(frame.content);
        prevTimestamp = frame.timestamp;

        setStatus(``);
      }

      if (!signal.aborted) {
        setStatus("Playback complete");
      }
    } catch (error: any) {
      if (error.name === "AbortError" || error.message === "Aborted") {
        setStatus("Playback stopped");
      } else {
        setStatus(`Error: ${error.message}`);
        console.error("Playback error:", error);
      }
    } finally {
      setIsPlaying(false);
      abortController.current = null;
    }
  };

  const stopPlayback = () => {
    if (abortController.current) {
      abortController.current.abort();
    }
  };

  const resetTerminal = () => {
    if (terminal.current) {
      terminal.current.clear();

      /*
      terminal.current.write(
        "SSH Session Replay - Press Play to start\r\n\r\n"
      );
      */
      setStatus("Ready to play");
    }
  };

  return (
    <div className="w-full h-full flex flex-col">
      <div
        ref={terminalRef}
        className="flex-1 border border-gray-700 rounded-lg shadow-2xl"
      />

      <div className="mt-4 flex items-center w-full bg-gray-900 shadow-lg rounded-lg p-4">
        <ActionIcon
          variant="transparent"
          aria-label="Play"
          onClick={() => {
            if (isPlaying) {
              stopPlayback();
            } else {
              playSession();
            }
          }}
        >
          {isPlaying ? (
            <FaStop size={28} color="white" />
          ) : (
            <FaPlay size={28} color="white" />
          )}
        </ActionIcon>
        <ActionIcon
          className="ml-3"
          variant="transparent"
          aria-label="Play"
          onClick={() => {
            resetTerminal();
          }}
        >
          <RiResetLeftFill size={28} color="white" />
        </ActionIcon>
        {/*
        <button
          onClick={playSession}
          disabled={isPlaying}
          className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed"
        >
          {isPlaying ? "Playing..." : "Play"}
        </button>
        <button
          onClick={stopPlayback}
          disabled={!isPlaying}
          className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 disabled:bg-gray-600 disabled:cursor-not-allowed"
        >
          Stop
        </button>
        <button
          onClick={resetTerminal}
          disabled={isPlaying}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed"
        >
          Reset
        </button>
       */}
        <div className="flex-1"></div>
        <span className="text-gray-300 ml-4 text-sm font-bold">{status}</span>
      </div>
    </div>
  );
};
