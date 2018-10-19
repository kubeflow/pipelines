import * as React from 'react';

export default class StatusRunning extends React.Component<{ color: string }> {
  public render() {
    return (
      <svg width='14px' height='14px' viewBox='0 0 14 14' version='1.1'>
        <g id='mlp' stroke='none' strokeWidth='1' fill='none' fillRule='evenodd'>
          <g id='400_jobs' transform='translate(-452.000000, -309.000000)' fill='#0F9D58' fillRule='nonzero'>
            <g id='Group-17' transform='translate(450.000000, 266.000000)'>
              <g id='Group-16'>
                <g id='Group-15'>
                  <g id='status-running' transform='translate(0.000000, 41.000000)'>
                    <path d='M9,4 C6.23857143,4 4,6.23857143 4,9 C4,11.7614286 6.23857143,14 9,14 C11.7614286,14 14,11.7614286 14,9 C14,8.40214643 13.8950716,7.8288007 13.702626,7.29737398 L15.2180703,5.78192967 C15.7177126,6.74539838 16,7.83973264 16,9 C16,12.866 12.866,16 9,16 C5.134,16 2,12.866 2,9 C2,5.134 5.134,2 9,2 C10.933,2 12.683,2.7835 13.94975,4.05025 L12.7677679,5.23223214 L9,9 L9,4 Z' id='Shape' />
                  </g>
                </g>
              </g>
            </g>
          </g>
        </g>
      </svg>
    );
  }
}
