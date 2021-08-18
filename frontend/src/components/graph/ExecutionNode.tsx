import DoneIcon from '@material-ui/icons/Done';
import React from 'react';

function ExecutionNode() {
  return (
    <div className='container mx-auto'>
      <div className='flex justify-between flex-row transform hover:scale-105 transition overflow:hidden relative mx-auto sm:mx-auto bg-white shadow-lg sm:rounded-xl sm:w-72'>
        <div className='sm:px-6 sm:py-4 sm:pb-3 w-60'>This is an execution</div>
        <div
          className='sm:px-4 sm:pt-4 sm:pb-3 sm:rounded-r-xl'
          style={{ backgroundColor: '#e6f4ea' }}
        >
          <DoneIcon style={{ color: '#34a853' }} />
        </div>
      </div>
    </div>
  );
}

export default ExecutionNode;
