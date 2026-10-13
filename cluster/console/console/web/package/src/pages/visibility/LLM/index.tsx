import LLMVisibility from "@/components/LLMVisibility";

const LLMVisibilityPage = () => (
  <div className="flex w-full flex-col gap-4 py-4">
    <LLMVisibility
      principalKinds={["user", "session", "service", "policy"]}
    />
  </div>
);

export default LLMVisibilityPage;
