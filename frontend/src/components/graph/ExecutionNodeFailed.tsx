import React from 'react';
import ClearIcon from '@material-ui/icons/Clear';
interface ExecutionNodeFailedProps {}

function ExecutionNodeFailed({}: ExecutionNodeFailedProps) {
  return (
    <div className='container mx-auto'>
      <div className='flex justify-between flex-row transform hover:scale-105 transition overflow:hidden relative mx-auto sm:mx-auto bg-white shadow-lg sm:rounded-xl sm:w-72'>
        <div className='sm:px-6 sm:py-4 sm:pb-3 w-60'>Failed execution</div>
        <div className='sm:px-4 sm:pt-4 sm:pb-3 sm:rounded-r-xl bg-red-200'>
          <ClearIcon style={{ color: '#FF0000' }} />
        </div>
      </div>
    </div>
  );
}

export default ExecutionNodeFailed;
