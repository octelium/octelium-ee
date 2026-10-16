const Footer = () => {
  return (
    <>
      <div className="w-full mb-6 pt-8">
        <div className="w-full px-2">
          <div className="w-full flex items-center justify-center">
            <a
              href="https://octelium.com"
              target="_blank"
              rel="noreferrer noopener"
            >
              <span className="text-xs font-medium text-slate-500 transition-colors duration-150 hover:text-slate-800">
                © {new Date().getFullYear()}{" "}
                <span className="ml-1">octelium.com</span>
              </span>
            </a>
          </div>
        </div>
      </div>
    </>
  );
};

export default Footer;
