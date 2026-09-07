import ResultClient from "@/components/ResultClient";

export default async function ResultPage({ params }: { params: Promise<{ attemptId: string }> }) {
  const { attemptId } = await params;
  return <ResultClient attemptID={attemptId}/>;
}
