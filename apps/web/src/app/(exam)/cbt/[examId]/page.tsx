import ExamClient from "@/components/ExamClient";

export default async function CBTExamPage({ params, searchParams }: { params: Promise<{ examId: string }>; searchParams: Promise<{ token?: string }> }) {
  const { examId } = await params;
  const { token } = await searchParams;
  return <ExamClient examID={examId} initialToken={token} />;
}
