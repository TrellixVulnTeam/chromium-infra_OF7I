import React from 'react';
import {makeStyles, withStyles} from '@material-ui/styles';
import {blue, yellow, red, grey} from '@material-ui/core/colors';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Checkbox, {CheckboxProps} from '@material-ui/core/Checkbox';
import SelectMenu from './SelectMenu.tsx';
import RadioDescription from './RadioDescription.tsx';

const CustomCheckbox = withStyles({
  root: {
    color: blue[400],
    '&$checked': {
      color: blue[600],
    },
  },
  checked: {},
})((props: CheckboxProps) => <Checkbox color="default" {...props} />);

const useStyles = makeStyles({
  pad: {
    margin: '10px, 20px',
    display: 'inline-block',
  },
  flex: {
    display: 'flex',
  },
  inlineBlock: {
    display: 'inline-block',
  },
  warningBox: {
    minHeight: '10vh',
    borderStyle: 'solid',
    borderWidth: '2px',
    borderColor: yellow[800],
    borderRadius: '8px',
    background: yellow[50],
    padding: '0px 20px 1em',
    margin: '30px 0px'
  },
  warningHeader: {
    color: yellow[800],
    fontSize: '16px',
    fontWeight: '500',
  },
  star:{
    color: red[700],
    marginRight: '8px',
    fontSize: '16px',
    display: 'inline-block',
  },
  header: {
    color: grey[900],
    fontSize: '28px',
    marginTop: '6vh',
  },
  subheader: {
    color: grey[700],
    fontSize: '18px',
    lineHeight: '32px',
  },
  red: {
    color: red[600],
  },
});

export default function LandingStep({checkExisting, setCheckExisting, userType, setUserType, category, setCategory}:
  {checkExisting: boolean, setCheckExisting: Function, userType: string, setUserType: Function, category: string, setCategory: Function}) {
  const classes = useStyles();

  const handleCheckChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setCheckExisting(event.target.checked);
  };

  return (
    <>
      <p className={classes.header}>Report an issue with Chromium</p>
      <p className={classes.subheader}>
        We want you to enter the best possible issue report so that the project team members
        can act on it effectively. The following steps will help route your issue to the correct
        people.
      </p>
      <p className={classes.subheader}>
        Please select your following role: <span className={classes.red}>*</span>
      </p>
      <RadioDescription value={userType} setValue={setUserType}/>
      <div className={classes.subheader}>
        Which of the following best describes the issue that you are reporting? <span className={classes.red}>*</span>
      </div>
      <SelectMenu option={category} setOption={setCategory}/>
      <div className={classes.warningBox}>
        <p className={classes.warningHeader}> Avoid duplicate issue reports:</p>
        <div>
          <div className={classes.star}>*</div>
          <FormControlLabel className={classes.pad}
            control={
              <CustomCheckbox
                checked={checkExisting}
                onChange={handleCheckChange}
                name="warningCheck"
              />
            }
            label="By checking this box, I'm acknowledging that I have searched for existing issues that already report this problem."
          />
        </div>
      </div>
    </>
  );
}