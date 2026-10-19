import LLMVisibility from "@/components/LLMVisibility";
import Meta from "@/components/Meta";

const LLMVisibilityPage = () => (
  <div className="flex w-full flex-col gap-4 py-4">
    <Meta title="LLM" />
    <LLMVisibility
      principalKinds={["user", "session", "service", "policy"]}
    />
  </div>
);

export default LLMVisibilityPage;
