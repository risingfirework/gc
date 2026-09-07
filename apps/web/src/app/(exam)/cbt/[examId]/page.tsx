import ExamClient from "@/components/ExamClient";

export default async function CBTExamPage({ params }: { params: Promise<{ examId: string }> }) {
  const { examId } = await params;
  return <ExamClient examID={examId} />;
}
