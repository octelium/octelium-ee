import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { SSHSession } from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import { Play, RotateCcw, Square } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { twMerge } from "tailwind-merge";

const MAX_FRAME_DELAY_MS = 2000;

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
}

const generateMockPage = (page: number, fromMs?: number): APIResponse => {
  const encoder = new TextEncoder();
  const baseTime = 1000000000000;

  const allFrames: { text: string; delay: number }[] = [
    { text: "\x1b[32muser@server:~$\x1b[0m ", delay: 0 },
    { text: "l", delay: 150 },
    { text: "s -la\r\n", delay: 200 },
    { text: "total 48\r\n", delay: 50 },
    {
      text: "drwxr-xr-x  5 user user 4096 Dec 17 10:30 \x1b[34m.\x1b[0m\r\n",
      delay: 20,
    },
    {
      text: "drwxr-xr-x  3 root root 4096 Dec 10 08:15 \x1b[34m..\x1b[0m\r\n",
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
    { text: "\x1b[32muser@server:~$\x1b[0m ", delay: 100 },
    { text: "cd Documents\r\n", delay: 2000 },
    { text: "\x1b[32muser@server:~/Documents$\x1b[0m ", delay: 50 },
    { text: "cat readme.txt\r\n", delay: 1500 },
    { text: "=================================\r\n", delay: 30 },
    { text: "Welcome to the SSH Replay Demo!\r\n", delay: 30 },
    { text: "=================================\r\n", delay: 30 },
    { text: "\x1b[32muser@server:~/Documents$\x1b[0m ", delay: 200 },
    { text: "exit\r\n", delay: 500 },
    { text: "\x1b[33mConnection to server closed.\x1b[0m\r\n", delay: 50 },
  ];

  let currentTime = baseTime;
  const timestampedFrames: SSHFrame[] = allFrames.map((item) => {
    currentTime += item.delay;
    return { content: encoder.encode(item.text), timestamp: currentTime };
  });

  const PAGE_SIZE = 6;
  const filtered =
    fromMs != null
      ? timestampedFrames.filter((f) => f.timestamp >= fromMs)
      : timestampedFrames;

  const start = (page - 1) * PAGE_SIZE;
  const end = start + PAGE_SIZE;
  const pageFrames = filtered.slice(start, end);
  const hasMore = end < filtered.length;

  return {
    frames: pageFrames,
    hasMore,
    nextPage: hasMore ? page + 1 : undefined,
  };
};

type PlaybackStatus =
  | { type: "idle" }
  | { type: "loading"; page: number }
  | { type: "playing"; current: number; total: number }
  | { type: "done" }
  | { type: "stopped" }
  | { type: "error"; message: string };

const statusLabel = (s: PlaybackStatus): string =>
  s.type === "idle"
    ? "Ready"
    : s.type === "loading"
      ? `Loading page ${s.page}…`
      : s.type === "playing"
        ? `Frame ${s.current} / ${s.total}`
        : s.type === "done"
          ? "Playback complete"
          : s.type === "stopped"
            ? "Stopped"
            : `Error: ${s.message}`;

const statusColor = (s: PlaybackStatus): string =>
  s.type === "error"
    ? "text-red-400"
    : s.type === "done"
      ? "text-emerald-400"
      : s.type === "stopped"
        ? "text-amber-400"
        : "text-slate-400";

const formatTime = (ms: number): string => {
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
};

export const XTermSSHReplay: React.FC<XTermSSHReplayProps> = ({
  initialPage = 1,
  sshSession,
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const fitAddon = useRef<FitAddon | null>(null);
  const abortController = useRef<AbortController | null>(null);

  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackStatus, setPlaybackStatus] = useState<PlaybackStatus>({
    type: "idle",
  });

  // Progress scrubbing state — all in milliseconds relative to session start
  const [durationMs, setDurationMs] = useState(0);
  const [positionMs, setPositionMs] = useState(0);
  const [isScrubbing, setIsScrubbing] = useState(false);
  const sessionStartMs = useRef<number>(0);

  useEffect(() => {
    if (!terminalRef.current) return;

    terminal.current = new Terminal({
      convertEol: true,
      fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, 'Ubuntu Mono', Consolas, monospace",
      fontWeight: 400,
      fontWeightBold: 700,
      cursorBlink: false,
      scrollback: 5000,
      fontSize: 13,
      lineHeight: 1.4,
      theme: {
        background: "#0d1117",
        foreground: "#c9d1d9",
        cursor: "#c9d1d9",
        selectionBackground: "#264f78",
        black: "#484f58",
        red: "#ff7b72",
        green: "#3fb950",
        yellow: "#d29922",
        blue: "#58a6ff",
        magenta: "#bc8cff",
        cyan: "#39c5cf",
        white: "#b1bac4",
        brightBlack: "#6e7681",
        brightRed: "#ffa198",
        brightGreen: "#56d364",
        brightYellow: "#e3b341",
        brightBlue: "#79c0ff",
        brightMagenta: "#d2a8ff",
        brightCyan: "#56d4dd",
        brightWhite: "#f0f6fc",
      },
    });

    fitAddon.current = new FitAddon();
    terminal.current.loadAddon(fitAddon.current);
    terminal.current.open(terminalRef.current);
    fitAddon.current.fit();

    const handleResize = () => fitAddon.current?.fit();
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      terminal.current?.dispose();
    };
  }, []);

  const fetchPage = useCallback(
    async (
      page: number,
      signal: AbortSignal,
      fromMs?: number,
    ): Promise<APIResponse> => {
      if (isDev()) {
        await new Promise((resolve) => setTimeout(resolve, 150));
        if (signal.aborted) throw new DOMException("Aborted", "AbortError");
        return generateMockPage(page, fromMs);
      }

      const { response } =
        await getClientVisibilityAccessLog().listSSHSessionRecording({
          sessionID: sshSession.id,
          page: page - 1,
          from:
            fromMs != null ? Timestamp.fromDate(new Date(fromMs)) : undefined,
        });

      return {
        frames: response.items.map((x) => ({
          content: x.data,
          timestamp: Timestamp.toDate(x.timestamp!).getTime(),
        })),
        hasMore: response.listResponseMeta?.hasMore ?? false,
        nextPage: response.listResponseMeta?.hasMore ? page + 1 : undefined,
      };
    },
    [sshSession.id],
  );

  const runPlayback = useCallback(
    async (fromMs?: number) => {
      if (!terminal.current) return;

      abortController.current?.abort();
      abortController.current = new AbortController();
      const signal = abortController.current.signal;

      setIsPlaying(true);
      terminal.current.clear();

      try {
        let currentPage = initialPage;
        const allFrames: SSHFrame[] = [];

        while (true) {
          if (signal.aborted) break;
          setPlaybackStatus({ type: "loading", page: currentPage });

          const response = await fetchPage(currentPage, signal, fromMs);
          allFrames.push(...response.frames);

          if (!response.hasMore || !response.nextPage) break;
          currentPage = response.nextPage;
        }

        if (signal.aborted) {
          setPlaybackStatus({ type: "stopped" });
          return;
        }

        if (allFrames.length === 0) {
          setPlaybackStatus({ type: "done" });
          return;
        }

        allFrames.sort((a, b) => a.timestamp - b.timestamp);

        const firstTs = allFrames[0].timestamp;
        const lastTs = allFrames[allFrames.length - 1].timestamp;

        // On first play (no seek) establish the session start and total duration
        if (fromMs == null) {
          sessionStartMs.current = firstTs;
          setDurationMs(lastTs - firstTs);
        }

        let prevTimestamp = firstTs;

        for (let i = 0; i < allFrames.length; i++) {
          if (signal.aborted) break;

          const frame = allFrames[i];
          const rawDelay = frame.timestamp - prevTimestamp;
          const delay = Math.min(rawDelay, MAX_FRAME_DELAY_MS);

          if (delay > 0 && i > 0) {
            await new Promise<void>((resolve, reject) => {
              const id = setTimeout(resolve, delay);
              signal.addEventListener(
                "abort",
                () => {
                  clearTimeout(id);
                  reject(new DOMException("Aborted", "AbortError"));
                },
                { once: true },
              );
            });
          }

          terminal.current!.write(frame.content);
          prevTimestamp = frame.timestamp;

          const elapsed = frame.timestamp - sessionStartMs.current;
          if (!isScrubbing) {
            setPositionMs(elapsed);
          }

          setPlaybackStatus({
            type: "playing",
            current: i + 1,
            total: allFrames.length,
          });
        }

        setPlaybackStatus(
          signal.aborted ? { type: "stopped" } : { type: "done" },
        );
        if (!signal.aborted) {
          setPositionMs(durationMs);
        }
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === "AbortError") {
          setPlaybackStatus({ type: "stopped" });
        } else {
          const message = err instanceof Error ? err.message : "Unknown error";
          setPlaybackStatus({ type: "error", message });
          console.error("Playback error:", err);
        }
      } finally {
        setIsPlaying(false);
        abortController.current = null;
      }
    },
    [initialPage, fetchPage, isScrubbing, durationMs],
  );

  const handlePlayPause = useCallback(() => {
    if (isPlaying) {
      abortController.current?.abort();
    } else {
      runPlayback();
    }
  }, [isPlaying, runPlayback]);

  const resetTerminal = useCallback(() => {
    abortController.current?.abort();
    terminal.current?.clear();
    setPlaybackStatus({ type: "idle" });
    setPositionMs(0);
    setDurationMs(0);
    sessionStartMs.current = 0;
  }, []);

  const handleScrubStart = useCallback(() => {
    setIsScrubbing(true);
    if (isPlaying) {
      abortController.current?.abort();
    }
  }, [isPlaying]);

  const inputRef = useRef<HTMLInputElement>(null);

  const handleScrubEnd = useCallback(() => {
    const targetMs = Number(inputRef.current?.value ?? 0);
    setPositionMs(targetMs);
    setIsScrubbing(false);
    const absoluteMs = sessionStartMs.current + targetMs;
    runPlayback(absoluteMs);
  }, [runPlayback]);

  const handleScrubMove = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (isScrubbing) {
        setPositionMs(Number(e.target.value));
      }
    },
    [isScrubbing],
  );

  const progressPct =
    durationMs > 0 ? Math.round((positionMs / durationMs) * 100) : 0;

  return (
    <div className="w-full flex flex-col rounded-xl overflow-hidden border border-slate-700 shadow-[0_4px_24px_rgba(1,4,9,0.4)]">
      <div
        ref={terminalRef}
        className="flex-1 min-h-[400px]"
        style={{ background: "#0d1117" }}
      />

      <div className="bg-[#161b22] border-t border-slate-700 px-4 pt-2.5 pb-3 flex flex-col gap-2">
        {/* Progress bar */}
        <div className="flex items-center gap-2">
          <span className="text-[0.65rem] font-semibold font-mono text-slate-500 w-10 shrink-0 text-right tabular-nums">
            {formatTime(positionMs)}
          </span>

          <div className="relative flex-1 flex items-center">
            <div className="absolute w-full h-1 bg-slate-700 rounded-full pointer-events-none">
              <div
                className="h-full bg-emerald-500 rounded-full transition-[width] duration-100"
                style={{ width: `${progressPct}%` }}
              />
            </div>
            <input
              ref={inputRef}
              type="range"
              min={0}
              max={durationMs || 100}
              step={100}
              value={positionMs}
              disabled={durationMs === 0}
              onMouseDown={handleScrubStart}
              onTouchStart={handleScrubStart}
              onChange={handleScrubMove}
              onMouseUp={handleScrubEnd}
              onTouchEnd={handleScrubEnd}
              className="relative w-full h-1 appearance-none bg-transparent cursor-pointer disabled:cursor-default disabled:opacity-40"
              style={{
                WebkitAppearance: "none",
              }}
            />
          </div>

          <span className="text-[0.65rem] font-semibold font-mono text-slate-500 w-10 shrink-0 tabular-nums">
            {formatTime(durationMs)}
          </span>
        </div>

        {/* Controls row */}
        <div className="flex items-center gap-2">
          <button
            onClick={handlePlayPause}
            className={twMerge(
              "flex items-center justify-center w-8 h-8 rounded-md cursor-pointer transition-colors duration-150",
              isPlaying
                ? "text-red-400 hover:text-red-300 hover:bg-red-400/10"
                : "text-emerald-400 hover:text-emerald-300 hover:bg-emerald-400/10",
            )}
            title={isPlaying ? "Stop" : "Play"}
          >
            {isPlaying ? (
              <Square size={14} strokeWidth={2.5} />
            ) : (
              <Play size={14} strokeWidth={2.5} />
            )}
          </button>

          <button
            onClick={resetTerminal}
            className="flex items-center justify-center w-8 h-8 rounded-md cursor-pointer text-slate-400 hover:text-slate-200 hover:bg-slate-400/10 transition-colors duration-150"
            title="Reset"
          >
            <RotateCcw size={13} strokeWidth={2.5} />
          </button>

          <div className="flex-1" />

          <span
            className={twMerge(
              "text-[0.72rem] font-semibold font-mono",
              statusColor(playbackStatus),
            )}
          >
            {statusLabel(playbackStatus)}
          </span>
        </div>
      </div>
    </div>
  );
};
