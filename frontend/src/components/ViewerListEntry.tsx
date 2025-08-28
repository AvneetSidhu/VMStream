import React from "react";

interface ListEntryProps {
  viewerName: string;
}

const ViewerListEntry: React.FC<ListEntryProps> = ({
  viewerName,
}: {
  viewerName: string;
}) => {
  return (
    <div className="bg-blue-400 text-white px-3 py-1 my-1 rounded shadow-sm">
      {viewerName}
    </div>
  );
};

export default ViewerListEntry;
