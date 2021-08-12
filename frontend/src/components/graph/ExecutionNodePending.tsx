import React from 'react';
import DragHandleIcon from '@material-ui/icons/DragHandle';
interface ExecutionNodePendingProps {}

function ExecutionNodePending({}: ExecutionNodePendingProps) {
  return (
    <div className='container mx-auto'>
      <div className='flex justify-between flex-row transform hover:scale-105 transition overflow:hidden relative mx-auto sm:mx-auto bg-white shadow-lg sm:rounded-xl sm:w-72'>
        <div className='sm:px-6 sm:py-4 sm:pb-3 w-60'>Pending execution</div>
        <div className='sm:px-4 sm:pt-4 sm:pb-3 sm:rounded-r-xl bg-gray-200'>
          <DragHandleIcon style={{ color: '#4B5563' }} />
        </div>
      </div>
    </div>
  );
}

export default ExecutionNodePending;
