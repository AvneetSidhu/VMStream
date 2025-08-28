import React from "react";
import ViewerListEntry from "./ViewerListEntry";

interface ViewerListProps {
  viewers: string[];
}

const ViewerList: React.FC<ViewerListProps> = ({ viewers }) => {
  return (
    <div>
      <h2 className="text-lg font-bold mb-4">Connected Viewers: </h2>
      {viewers.map((viewer) => (
        <ViewerListEntry key={viewer} viewerName={viewer} />
      ))}
    </div>
  );
};

export default ViewerList;
