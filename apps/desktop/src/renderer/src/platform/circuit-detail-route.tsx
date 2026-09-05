import { useParams } from "react-router-dom";
import { CircuitDetailPage } from "@chimii/views/circuit";
export function CircuitDetailRoute() {
  const { id } = useParams<{ id: string }>();
  return id ? <CircuitDetailPage creationId={id} /> : null;
}
