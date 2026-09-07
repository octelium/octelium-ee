export default () => (
  <div className="flex w-full items-center justify-center pb-6 pt-10">
    <a
      href="https://octelium.com"
      target="_blank"
      rel="noreferrer noopener"
      className="text-xs font-normal text-slate-500 transition-colors duration-150 hover:text-slate-800"
    >
      © {new Date().getFullYear()} octelium.com
    </a>
  </div>
);
