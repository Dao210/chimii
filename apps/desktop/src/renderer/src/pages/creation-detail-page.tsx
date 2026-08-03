import { useParams } from "react-router-dom";
import { CreationDetailPage as CreationDetailPageView } from "@chimii/views/build";

export function CreationDetailPage() {
  const { id } = useParams<{ id: string }>();
  if (!id) return null;
  return <CreationDetailPageView creationId={id} />;
}
